package adapters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shouni/go-remote-io/remoteio"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

type MusicRepository struct {
	cfg    *config.Config
	reader remoteio.Reader
}

// NewGCSMusicRepository はリポジトリを初期化するのだ。
func NewGCSMusicRepository(cfg *config.Config, reader remoteio.Reader) *MusicRepository {
	return &MusicRepository{
		cfg:    cfg,
		reader: reader,
	}
}

// ListHistory は、GCSのファイル一覧を取得して MusicHistory のリストを作成します。
func (r *MusicRepository) ListHistory(ctx context.Context, userID string) ([]domain.MusicHistory, error) {
	// 1. バケット内のファイルをリストアップするのだ（本来は userID でフィルタリング）
	// 2. ".json" で終わるファイルを探すのだ
	// 3. ファイル名から MusicHistory 構造体を組み立てて返すのだ
	return nil, nil // ここに具体的な List ロジックを書くのだ！
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
