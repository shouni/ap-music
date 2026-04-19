package adapters

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"unicode/utf8"

	"github.com/shouni/go-manga-kit/ports"
	"github.com/shouni/go-remote-io/remoteio"
	"github.com/shouni/go-web-reader/pkg/reader"

	"ap-music/internal/domain"
)

const (
	// maxInputSize は読み込みを許可する最大テキストサイズ (5MB) です。
	maxInputSize = 5 * 1024 * 1024
)

// ReaderAdapter は入力情報を収集します。
type ReaderAdapter struct {
	contentReader ports.ContentReader
}

func NewReaderAdapter(storage remoteio.ReadWriteFactory) (*ReaderAdapter, error) {
	contentReader, err := reader.New(
		reader.WithGCSFactory(func(ctx context.Context) (remoteio.ReadWriteFactory, error) {
			return storage, nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize content reader: %w", err)
	}

	return &ReaderAdapter{
		contentReader: contentReader,
	}, nil
}

// Collect は、コンテンツを取得します。
func (r *ReaderAdapter) Collect(ctx context.Context, task domain.Task) (string, error) {
	return r.readContent(ctx, task)
}

// readContent は、指定されたソースURLからコンテンツを取得します。
func (r *ReaderAdapter) readContent(ctx context.Context, task domain.Task) (string, error) {
	url := task.RequestURL
	rc, err := r.contentReader.Open(ctx, url)
	if err != nil {
		return "", fmt.Errorf("failed to read source: %w", err)
	}
	defer func() {
		if closeErr := rc.Close(); closeErr != nil {
			slog.WarnContext(ctx, "ストリームのクローズに失敗しました", "error", closeErr)
		}
	}()
	limitedReader := io.LimitReader(rc, int64(maxInputSize))
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("読み込みに失敗しました: %w", err)
	}

	// 追加の読み込みを試みて切り捨てを判定
	oneMoreByte := make([]byte, 1)
	n, readErr := rc.Read(oneMoreByte)
	if readErr != nil && readErr != io.EOF {
		return "", fmt.Errorf("サイズ確認中にエラーが発生しました: %w", readErr)
	}

	if n > 0 {
		slog.WarnContext(ctx, "制限サイズに達したため切り捨てられました",
			"url", url,
			"limit_bytes", maxInputSize)

		// UTF-8の文字境界に合わせて末尾の不完全なバイトを取り除く
		for len(content) > 0 {
			r, size := utf8.DecodeLastRune(content)
			if r == utf8.RuneError && size == 1 {
				content = content[:len(content)-1]
			} else {
				break
			}
		}
	}

	return string(content), nil
}
