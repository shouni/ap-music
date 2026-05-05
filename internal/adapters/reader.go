package adapters

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-remote-io/remoteio"
	"github.com/shouni/go-web-reader/pkg/reader"

	"ap-music/internal/domain"
)

const (
	// maxInputSize は読み込みを許可する最大テキストサイズ (5MB) です。
	maxInputSize = 5 * 1024 * 1024
)

// ContentReader は、指定されたURIからコンテンツを取得するためのインターフェースです。
type ContentReader interface {
	Open(ctx context.Context, uri string) (io.ReadCloser, error)
}

// ReaderAdapter は入力情報を収集します。
type ReaderAdapter struct {
	contentReader ContentReader
	requester     httpkit.Requester
}

func NewReaderAdapter(storage remoteio.IOFactory) (*ReaderAdapter, error) {
	contentReader, err := reader.New(
		reader.WithGCSFactory(func(ctx context.Context) (remoteio.IOFactory, error) {
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
func (r *ReaderAdapter) Collect(ctx context.Context, task domain.Task) (*domain.CollectedContent, error) {
	res := &domain.CollectedContent{}
	var textParts []string
	if err := task.ValidateSubmission(); err != nil {
		return nil, err
	}

	var parts []string

	requestURL := strings.TrimSpace(task.RequestURL)
	if requestURL != "" {
		content, err := r.readURLContent(ctx, requestURL)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(content) == "" {
			return nil, fmt.Errorf("source URL content is empty: %s", requestURL)
		}
		parts = append(parts, fmt.Sprintf("[Source URL]\n%s\n\n[Source Content]\n%s", requestURL, content))
	}

	inputText := strings.TrimSpace(task.InputText)
	if inputText != "" {
		parts = append(parts, fmt.Sprintf("[User Input]\n%s", inputText))
	}

	imageURL := strings.TrimSpace(task.ImageURL)
	if imageURL != "" {
		imgData, err := r.requester.FetchBytes(ctx, imageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch image from %s: %w", imageURL, err)
		}
		mimeType := http.DetectContentType(imgData)
		if !strings.HasPrefix(mimeType, "image/") {
			return nil, fmt.Errorf("unsupported file type: %s", mimeType)
		}

		res.Images = append(res.Images, domain.ImagePayload{
			Data:     imgData,
			MIMEType: mimeType,
		})
	}

	res.Prompt = strings.Join(textParts, "\n\n")
	return res, nil
}

// readURLContent は、指定されたソースURLからコンテンツを取得します。
func (r *ReaderAdapter) readURLContent(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("request URL is empty")
	}
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
