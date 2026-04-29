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
	gcsPrefix := r.cfg.GCSBucket
	var histories []domain.MusicHistory

	err := r.reader.List(ctx, gcsPrefix, func(gcsPath string) error {
		if !strings.HasSuffix(gcsPath, ".json") {
			return nil
		}
		fileName := path.Base(gcsPath)
		jobID := strings.TrimSuffix(fileName, ".json")
		if jobID == "" {
			return nil
		}
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
