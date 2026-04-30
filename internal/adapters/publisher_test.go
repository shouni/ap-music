package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"
	"unicode/utf8"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

type stubWriter struct {
	writes       []string
	dataByURI    map[string][]byte
	contentTypes map[string]string
	failOn       map[string]error
}

func (w *stubWriter) Write(_ context.Context, uri string, contentReader io.Reader, contentType string) error {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, contentReader); err != nil {
		return err
	}
	w.writes = append(w.writes, uri)
	if w.dataByURI == nil {
		w.dataByURI = make(map[string][]byte)
	}
	if w.contentTypes == nil {
		w.contentTypes = make(map[string]string)
	}
	w.dataByURI[uri] = append([]byte(nil), buf.Bytes()...)
	w.contentTypes[uri] = contentType
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

func TestPublisherPublishWritesRecipeJSONAsUTF8(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	recipeURI := "gs://bucket/job-utf8.json"

	writer := &stubWriter{}
	signer := &testURLSigner{}
	cleaner := &stubCleaner{}

	publisher, err := NewPublisherAdapter(cfg, writer, signer, cleaner)
	if err != nil {
		t.Fatalf("NewPublisherAdapter() error = %v", err)
	}

	recipe := &domain.MusicRecipe{
		Title: "朝焼けのレシピ",
		Theme: "日本語ボーカル",
		Mood:  "明るい",
		Sections: []domain.MusicSection{
			{Name: "サビ", Duration: 30, Prompt: "透明感のある歌声"},
		},
	}
	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-utf8"}, recipe, []byte("wav"))
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	recipeData := writer.dataByURI[recipeURI]
	if writer.contentTypes[recipeURI] != recipeJSONContentType {
		t.Fatalf("recipe content type = %q, want %q", writer.contentTypes[recipeURI], recipeJSONContentType)
	}
	if !utf8.Valid(recipeData) {
		t.Fatalf("recipe json is not valid UTF-8: %q", recipeData)
	}
	if !json.Valid(recipeData) {
		t.Fatalf("recipe json is invalid: %s", recipeData)
	}
	if !bytes.Contains(recipeData, []byte("朝焼けのレシピ")) {
		t.Fatalf("recipe json does not contain raw UTF-8 title: %s", recipeData)
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
