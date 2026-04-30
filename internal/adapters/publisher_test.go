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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubWriter は remoteio.OutputWriter をシミュレートします
type stubWriter struct {
	writes       []string
	deletes      []string
	dataByURI    map[string][]byte
	contentTypes map[string]string
	failOn       map[string]error
}

func (w *stubWriter) Write(_ context.Context, uri string, contentReader io.Reader, contentType string) error {
	if err, ok := w.failOn[uri]; ok {
		return err
	}

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
	w.dataByURI[uri] = buf.Bytes()
	w.contentTypes[uri] = contentType
	return nil
}

func (w *stubWriter) Delete(_ context.Context, uri string) error {
	w.deletes = append(w.deletes, uri)
	return nil
}

// testURLSigner は remoteio.URLSigner をシミュレートします
type testURLSigner struct {
	calls  []string
	failOn map[string]error
}

func (s *testURLSigner) GenerateSignedURL(_ context.Context, uri string, _ string, _ time.Duration) (string, error) {
	if err, ok := s.failOn[uri]; ok {
		return "", err
	}
	s.calls = append(s.calls, uri)
	return "https://signed.example/" + uri, nil
}

func TestPublisherPublishCleansUpOnRecipeWriteFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	audioURI := "gs://bucket/job-1.wav"
	recipeURI := "gs://bucket/job-1.json"

	// レシピの書き込みで失敗するように設定
	writer := &stubWriter{failOn: map[string]error{
		recipeURI: errors.New("recipe write failed"),
	}}
	signer := &testURLSigner{}

	publisher, err := NewPublisherAdapter(cfg, writer, signer)
	require.NoError(t, err)

	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-1"}, &domain.MusicRecipe{Title: "x"}, []byte("wav"))
	assert.Error(t, err)

	// クリーンアップが呼ばれたことを確認 (recipeURI, audioURI の順)
	expectedDeletes := []string{recipeURI, audioURI}
	assert.Equal(t, expectedDeletes, writer.deletes)
}

func TestPublisherPublishCleansUpOnSignedURLFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	audioURI := "gs://bucket/job-2.wav"
	recipeURI := "gs://bucket/job-2.json"

	writer := &stubWriter{}
	// 署名URL生成で失敗するように設定
	signer := &testURLSigner{failOn: map[string]error{
		audioURI: errors.New("sign failed"),
	}}

	publisher, err := NewPublisherAdapter(cfg, writer, signer)
	require.NoError(t, err)

	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-2"}, &domain.MusicRecipe{Title: "x"}, []byte("wav"))
	assert.Error(t, err)

	// クリーンアップが呼ばれたことを確認
	expectedDeletes := []string{recipeURI, audioURI}
	assert.Equal(t, expectedDeletes, writer.deletes)
}

func TestPublisherPublishWritesRecipeJSONAsUTF8(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	recipeURI := "gs://bucket/job-utf8.json"

	writer := &stubWriter{}
	signer := &testURLSigner{}

	publisher, err := NewPublisherAdapter(cfg, writer, signer)
	require.NoError(t, err)

	recipe := &domain.MusicRecipe{
		Title: "朝焼けのレシピ",
		Theme: "日本語ボーカル",
	}
	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-utf8"}, recipe, []byte("wav"))
	require.NoError(t, err)

	recipeData := writer.dataByURI[recipeURI]

	// Content-Type の確認
	assert.Equal(t, recipeJSONContentType, writer.contentTypes[recipeURI])
	// JSONの妥当性とUTF-8の確認
	assert.True(t, utf8.Valid(recipeData))
	assert.True(t, json.Valid(recipeData))
	// エスケープされずに日本語が含まれているか（json.Marshalのデフォルト挙動に依存するが、ここではデータの存在を確認）
	assert.Contains(t, string(recipeData), "朝焼けのレシピ")
}
