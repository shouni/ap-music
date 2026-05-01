package repository

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"ap-music/internal/config"
)

type fakeHistoryReader struct {
	paths     []string
	files     map[string]string
	mu        sync.Mutex
	openCount map[string]int
}

func (r *fakeHistoryReader) Open(_ context.Context, path string) (io.ReadCloser, error) {
	r.mu.Lock()
	if r.openCount == nil {
		r.openCount = make(map[string]int)
	}
	r.openCount[path]++
	r.mu.Unlock()

	return io.NopCloser(strings.NewReader(r.files[path])), nil
}

func (r *fakeHistoryReader) List(_ context.Context, _ string, callback func(path string) error) error {
	for _, p := range r.paths {
		if err := callback(p); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeHistoryReader) Exists(context.Context, string) (bool, error) {
	return true, nil
}

func (r *fakeHistoryReader) countOpen(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.openCount[path]
}

type fakeHistoryWriter struct {
	mu      sync.Mutex
	deleted []string
}

func (w *fakeHistoryWriter) Write(context.Context, string, io.Reader, string) error {
	return nil
}

func (w *fakeHistoryWriter) Delete(_ context.Context, path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.deleted = append(w.deleted, path)
	return nil
}

func TestListHistoryLoadsRecipeMetadata(t *testing.T) {
	t.Parallel()

	reader := &fakeHistoryReader{
		paths: []string{
			"gs://music/20260501123456-abcd1234.json",
			"gs://music/ignore.wav",
		},
		files: map[string]string{
			"gs://music/20260501123456-abcd1234.json": `{
             "title": "テスト曲",
             "mood": "透明感",
             "tempo": 132,
             "compose_mode": "rave",
             "seed": 42
          }`,
		},
	}
	repo := NewGCSMusicRepository(&config.Config{GCSBucket: "music"}, reader, nil)

	histories, err := repo.ListHistory(context.Background(), "me")
	if err != nil {
		t.Fatalf("ListHistory() error = %v", err)
	}
	if len(histories) != 1 {
		t.Fatalf("len(histories) = %d, want 1", len(histories))
	}

	got := histories[0]
	if got.JobID != "20260501123456-abcd1234" {
		t.Fatalf("JobID = %q", got.JobID)
	}
	if got.Title != "テスト曲" {
		t.Fatalf("Title = %q", got.Title)
	}
	if got.Mood != "透明感" {
		t.Fatalf("Mood = %q", got.Mood)
	}
	if got.Tempo != 132 {
		t.Fatalf("Tempo = %d", got.Tempo)
	}
	if got.ComposeMode != "rave" {
		t.Fatalf("ComposeMode = %q", got.ComposeMode)
	}
	if got.Seed != "42" {
		t.Fatalf("Seed = %q", got.Seed)
	}
	if got.CreatedAt != "2026-05-01 12:34 UTC" {
		t.Fatalf("CreatedAt = %q", got.CreatedAt)
	}
}

func TestListHistoryUsesCachedMetadata(t *testing.T) {
	t.Parallel()

	const objectPath = "gs://music/20260501123456-abcd1234.json"
	reader := &fakeHistoryReader{
		paths: []string{objectPath},
		files: map[string]string{
			objectPath: `{"title":"初回タイトル","tempo":132}`,
		},
	}
	repo := NewGCSMusicRepository(&config.Config{GCSBucket: "music"}, reader, nil)

	if _, err := repo.ListHistory(context.Background(), "me"); err != nil {
		t.Fatalf("first ListHistory() error = %v", err)
	}
	reader.files[objectPath] = `{"title":"更新後タイトル","tempo":140}`
	histories, err := repo.ListHistory(context.Background(), "me")
	if err != nil {
		t.Fatalf("second ListHistory() error = %v", err)
	}

	if got := reader.countOpen(objectPath); got != 1 {
		t.Fatalf("Open count = %d, want 1", got)
	}
	if got := histories[0].Title; got != "初回タイトル" {
		t.Fatalf("cached Title = %q, want 初回タイトル", got)
	}
}

func TestDeleteHistoryInvalidatesCachedMetadata(t *testing.T) {
	t.Parallel()

	const objectPath = "gs://music/20260501123456-abcd1234.json"
	reader := &fakeHistoryReader{
		paths: []string{objectPath},
		files: map[string]string{
			objectPath: `{"title":"削除前タイトル","tempo":132}`,
		},
	}
	writer := &fakeHistoryWriter{}
	repo := NewGCSMusicRepository(&config.Config{GCSBucket: "music"}, reader, writer)

	if _, err := repo.ListHistory(context.Background(), "me"); err != nil {
		t.Fatalf("first ListHistory() error = %v", err)
	}
	if err := repo.DeleteHistory(context.Background(), "20260501123456-abcd1234"); err != nil {
		t.Fatalf("DeleteHistory() error = %v", err)
	}

	reader.files[objectPath] = `{"title":"削除後タイトル","tempo":140}`
	histories, err := repo.ListHistory(context.Background(), "me")
	if err != nil {
		t.Fatalf("second ListHistory() error = %v", err)
	}

	if got := reader.countOpen(objectPath); got != 2 {
		t.Fatalf("Open count = %d, want 2", got)
	}
	if got := histories[0].Title; got != "削除後タイトル" {
		t.Fatalf("Title after cache invalidation = %q, want 削除後タイトル", got)
	}
}
