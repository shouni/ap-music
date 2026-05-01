package repository

import (
	"context"
	"io"
	"strings"
	"testing"

	"ap-music/internal/config"
)

type fakeHistoryReader struct {
	paths []string
	files map[string]string
}

func (r fakeHistoryReader) Open(_ context.Context, path string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(r.files[path])), nil
}

func (r fakeHistoryReader) List(_ context.Context, _ string, callback func(path string) error) error {
	for _, p := range r.paths {
		if err := callback(p); err != nil {
			return err
		}
	}
	return nil
}

func (r fakeHistoryReader) Exists(context.Context, string) (bool, error) {
	return true, nil
}

func TestListHistoryLoadsRecipeMetadata(t *testing.T) {
	t.Parallel()

	reader := fakeHistoryReader{
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
