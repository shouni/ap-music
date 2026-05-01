package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	writer remoteio.OutputWriter
}

// NewGCSMusicRepository はリポジトリを初期化するのだ。
func NewGCSMusicRepository(cfg *config.Config, reader remoteio.InputReader, writer remoteio.OutputWriter) *MusicRepository {
	return &MusicRepository{
		cfg:    cfg,
		reader: reader,
		writer: writer,
	}
}

// ListHistory は、GCSのファイル一覧を取得して MusicHistory のリストを作成します。
func (r *MusicRepository) ListHistory(ctx context.Context, userID string) ([]domain.MusicHistory, error) {
	gcsURI := r.cfg.GetGCSObjectURL("")
	// バケット直下をリストする場合でも、末尾にスラッシュが必要
	if !strings.HasSuffix(gcsURI, "/") {
		gcsURI += "/"
	}
	var histories []domain.MusicHistory

	err := r.reader.List(ctx, gcsURI, func(gcsPath string) error {
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
	safeJobID := path.Base(jobID)
	objectPath := fmt.Sprintf("%s.json", safeJobID)
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

// DeleteHistory は、指定されたジョブIDに関連するJSONファイルとオーディオファイルを削除します。
func (r *MusicRepository) DeleteHistory(ctx context.Context, jobID string) error {
	safeJobID := path.Base(jobID)
	var errs []error

	// 1. レシピ JSON ファイルの削除
	jsonPath := fmt.Sprintf("%s.json", safeJobID)
	jsonURI := r.cfg.GetGCSObjectURL(jsonPath)
	if err := r.writer.Delete(ctx, jsonURI); err != nil {
		errs = append(errs, fmt.Errorf("failed to delete recipe JSON (%s): %w", jsonURI, err))
	}

	// 2. オーディオファイルの削除 (JSONの成否に関わらず実行する)
	audioPath := fmt.Sprintf("%s.wav", safeJobID)
	audioURI := r.cfg.GetGCSObjectURL(audioPath)
	if err := r.writer.Delete(ctx, audioURI); err != nil {
		slog.WarnContext(ctx, "skipped or failed to delete audio file",
			"jobID", safeJobID,
			"uri", audioURI,
			"error", err,
		)
	}

	return errors.Join(errs...)
}
