package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/shouni/go-remote-io/remoteio"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

type MusicRepository struct {
	cfg    *config.Config
	reader remoteio.InputReader
}

// NewGCSMusicRepository はリポジトリを初期化するのだ。
func NewGCSMusicRepository(cfg *config.Config, reader remoteio.InputReader) *MusicRepository {
	return &MusicRepository{
		cfg:    cfg,
		reader: reader,
	}
}

// ListHistory は、GCSのファイル一覧を取得して MusicHistory のリストを作成します。
func (r *MusicRepository) ListHistory(ctx context.Context, userID string) ([]domain.MusicHistory, error) {
	// 本来は userID ごとのディレクトリ分けなどを検討しますが、
	// 今回は config から取得した GCSPrefix 全体を対象にします。
	gcsPrefix := r.cfg.GCSBucket // もしくは特定のフォルダパス

	var histories []domain.MusicHistory

	// 1. バケット内のファイルをリストアップするのだ
	err := r.reader.List(ctx, gcsPrefix, func(gcsPath string) error {
		// 2. ".json" で終わるファイルを探すのだ
		if !strings.HasSuffix(gcsPath, ".json") {
			return nil
		}

		// ファイル名（パスの末尾）を取得
		fileName := path.Base(gcsPath)
		// 拡張子 .json を除いたものを JobID とみなす
		jobID := strings.TrimSuffix(fileName, ".json")

		// 3. ファイル名から MusicHistory 構造体を組み立てるのだ
		// ※ 本来は JSON の中身を少し覗いて Title などを取得するのが理想的ですが、
		//    パフォーマンスを優先してまずはファイル名から ID を抽出します。
		histories = append(histories, domain.MusicHistory{
			JobID: jobID,
			// Title などは別途 GetRecipe で取得するか、
			// ファイル名規則に Title を含める（timestamp-title-id.json等）と一覧性が上がります。
			Title: jobID,
		})
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("GCS履歴のリスト取得に失敗したのだ: %w", err)
	}

	// 降順（新しい順）にソートするのだ（JobID がタイムスタンプ開始と仮定）
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].JobID > histories[j].JobID
	})

	return histories, nil
}

// GetRecipe は、特定の JSON ファイルを読み込んで構造体にパースします。
func (r *MusicRepository) GetRecipe(ctx context.Context, jobID string) (*domain.MusicRecipe, error) {
	objectPath := fmt.Sprintf("%s.json", jobID)
	gcsURI := r.cfg.GetGCSObjectURL(objectPath)
	rc, err := r.reader.Open(ctx, gcsURI)
	if err != nil {
		return nil, fmt.Errorf("JSONオープン失敗 (%s): %w", gcsURI, err)
	}
	defer rc.Close()

	var recipe domain.MusicRecipe
	if err := json.NewDecoder(rc).Decode(&recipe); err != nil {
		return nil, fmt.Errorf("JSONデコード失敗: %w", err)
	}

	return &recipe, nil
}
