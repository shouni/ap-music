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
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"github.com/shouni/go-remote-io/remoteio"
	"golang.org/x/sync/errgroup"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

type MusicRepository struct {
	cfg          *config.Config
	reader       remoteio.InputReader
	writer       remoteio.OutputWriter
	historyCache *gocache.Cache
}

const defaultHistoryCacheTTL = 10 * time.Minute

func NewGCSMusicRepository(cfg *config.Config, reader remoteio.InputReader, writer remoteio.OutputWriter) *MusicRepository {
	return &MusicRepository{
		cfg:          cfg,
		reader:       reader,
		writer:       writer,
		historyCache: gocache.New(defaultHistoryCacheTTL, defaultHistoryCacheTTL),
	}
}

// ListHistory は並行処理を用いて履歴一覧を高速に取得します。
func (r *MusicRepository) ListHistory(ctx context.Context, userID string) ([]domain.MusicHistory, error) {
	gcsURI := r.cfg.GetGCSObjectURL("")
	if !strings.HasSuffix(gcsURI, "/") {
		gcsURI += "/"
	}

	// 1. まずファイル一覧（JobID）を取得する
	var jobIDs []string
	err := r.reader.List(ctx, gcsURI, func(gcsPath string) error {
		if !strings.HasSuffix(gcsPath, ".json") {
			return nil
		}
		fileName := path.Base(gcsPath)
		jobID := strings.TrimSuffix(fileName, ".json")
		if jobID != "" {
			jobIDs = append(jobIDs, jobID)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("GCS履歴のリスト取得に失敗したのだ: %w", err)
	}

	// 2. 並行して詳細（Recipe）を取得する
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(10)

	histories := make([]domain.MusicHistory, len(jobIDs))
	var mu sync.Mutex

	for i, id := range jobIDs {
		eg.Go(func() error {
			history, err := r.buildHistory(ctx, id)
			if err != nil {
				slog.WarnContext(ctx, "failed to load recipe metadata for history list",
					"jobID", id,
					"error", err,
				)
				// 取得失敗時はフォールバックデータを生成
				history = domain.MusicHistory{
					JobID:     id,
					Title:     id,
					CreatedAt: formatHistoryCreatedAt(id),
				}
			}

			mu.Lock()
			histories[i] = history
			mu.Unlock()
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	// 3. 最後にソート（新しい順）
	sort.Slice(histories, func(i, j int) bool {
		return histories[i].JobID > histories[j].JobID
	})

	return histories, nil
}

func (r *MusicRepository) buildHistory(ctx context.Context, jobID string) (domain.MusicHistory, error) {
	if history, ok := r.getCachedHistory(jobID); ok {
		return history, nil
	}

	recipe, err := r.GetRecipe(ctx, jobID)
	if err != nil {
		return domain.MusicHistory{}, err
	}

	title := strings.TrimSpace(recipe.Title)
	if title == "" {
		title = jobID
	}

	history := domain.MusicHistory{
		JobID:       jobID,
		Title:       title,
		Mood:        strings.TrimSpace(recipe.Mood),
		Tempo:       recipe.Tempo,
		CreatedAt:   formatHistoryCreatedAt(jobID),
		ComposeMode: strings.TrimSpace(recipe.ComposeMode),
	}
	if recipe.Seed != nil {
		history.Seed = fmt.Sprintf("%d", *recipe.Seed)
	}

	r.setCachedHistory(jobID, history)

	return history, nil
}

func (r *MusicRepository) getCachedHistory(jobID string) (domain.MusicHistory, bool) {
	value, ok := r.historyCache.Get(jobID)
	if !ok {
		return domain.MusicHistory{}, false
	}

	history, ok := value.(domain.MusicHistory)
	if !ok {
		r.historyCache.Delete(jobID)
		return domain.MusicHistory{}, false
	}

	return history, true
}

func (r *MusicRepository) setCachedHistory(jobID string, history domain.MusicHistory) {
	r.historyCache.SetDefault(jobID, history)
}

func (r *MusicRepository) deleteCachedHistory(jobID string) {
	r.historyCache.Delete(jobID)
}

// formatHistoryCreatedAt は、JobIDから日付を安全にパースします。
func formatHistoryCreatedAt(jobID string) string {
	const (
		jobIDTimePrefixLen = 14 // "20060102150405"
		jobIDTimeLayout    = "20060102150405"
		displayTimeLayout  = "2006-01-02 15:04 UTC"
	)

	if len(jobID) < jobIDTimePrefixLen {
		return ""
	}

	prefix := jobID[:jobIDTimePrefixLen]
	for _, char := range prefix {
		if char < '0' || char > '9' {
			return ""
		}
	}

	createdAt, err := time.ParseInLocation(jobIDTimeLayout, prefix, time.UTC)
	if err != nil {
		return ""
	}

	return createdAt.Format(displayTimeLayout)
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

// DeleteHistory は、関連ファイルを削除します。
func (r *MusicRepository) DeleteHistory(ctx context.Context, jobID string) error {
	safeJobID := path.Base(jobID)
	var errs []error

	jsonPath := fmt.Sprintf("%s.json", safeJobID)
	jsonURI := r.cfg.GetGCSObjectURL(jsonPath)
	if err := r.writer.Delete(ctx, jsonURI); err != nil {
		errs = append(errs, fmt.Errorf("failed to delete recipe JSON (%s): %w", jsonURI, err))
	}

	audioPath := fmt.Sprintf("%s.wav", safeJobID)
	audioURI := r.cfg.GetGCSObjectURL(audioPath)
	if err := r.writer.Delete(ctx, audioURI); err != nil {
		slog.WarnContext(ctx, "skipped or failed to delete audio file",
			"jobID", safeJobID,
			"uri", audioURI,
			"error", err,
		)
	}

	if err := errors.Join(errs...); err != nil {
		return err
	}

	r.deleteCachedHistory(safeJobID)
	return nil
}
