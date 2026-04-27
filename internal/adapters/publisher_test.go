package adapters

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

type stubWriter struct {
	writes []string
	failOn map[string]error
}

func (w *stubWriter) Write(_ context.Context, uri string, contentReader io.Reader, _ string) error {
	if _, err := io.Copy(io.Discard, contentReader); err != nil {
		return err
	}
	w.writes = append(w.writes, uri)
	if err, ok := w.failOn[uri]; ok {
		return err
	}
	return nil
}

type stubCleaner struct {
	deletes []string
}

func (c *stubCleaner) Delete(_ context.Context, uri string) error {
	c.deletes = append(c.deletes, uri)
	return nil
}

func TestPublisherPublishCleansUpOnRecipeWriteFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	audioURI := "gs://bucket/job-1.wav"
	recipeURI := "gs://bucket/job-1.json"

	writer := &stubWriter{failOn: map[string]error{
		recipeURI: errors.New("recipe write failed"),
	}}
	signer := &testURLSigner{}
	cleaner := &stubCleaner{}

	publisher, err := NewPublisherAdapter(cfg, writer, signer, cleaner)
	if err != nil {
		t.Fatalf("NewPublisherAdapter() error = %v", err)
	}

	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-1"}, &domain.MusicRecipe{Title: "x"}, []byte("wav"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if len(cleaner.deletes) != 2 || cleaner.deletes[0] != recipeURI || cleaner.deletes[1] != audioURI {
		t.Fatalf("unexpected cleanup order: %#v", cleaner.deletes)
	}
}

func TestPublisherPublishCleansUpOnSignedURLFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	audioURI := "gs://bucket/job-2.wav"
	recipeURI := "gs://bucket/job-2.json"

	writer := &stubWriter{}
	signer := &testURLSigner{failOn: map[string]error{
		audioURI: errors.New("sign failed"),
	}}
	cleaner := &stubCleaner{}

	publisher, err := NewPublisherAdapter(cfg, writer, signer, cleaner)
	if err != nil {
		t.Fatalf("NewPublisherAdapter() error = %v", err)
	}

	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-2"}, &domain.MusicRecipe{Title: "x"}, []byte("wav"))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if len(cleaner.deletes) != 2 || cleaner.deletes[0] != recipeURI || cleaner.deletes[1] != audioURI {
		t.Fatalf("unexpected cleanup order: %#v", cleaner.deletes)
	}
}

type testURLSigner struct {
	calls  []string
	failOn map[string]error
}

func (s *testURLSigner) GenerateSignedURL(_ context.Context, uri string, _ string, _ time.Duration) (string, error) {
	s.calls = append(s.calls, uri)
	if err, ok := s.failOn[uri]; ok {
		return "", err
	}
	return "https://signed.example/" + uri, nil
}
