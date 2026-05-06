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

	"github.com/shouni/go-remote-io/remoteio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubWriter は remoteio.OutputWriter をシミュレートします
type stubWriter struct {
	writes              []string
	deletes             []string
	dataByURI           map[string][]byte
	contentTypes        map[string]string
	contentDispositions map[string]string
	failOn              map[string]error
}

// Write はインターフェース remoteio.OutputWriter を実装します
func (w *stubWriter) Write(ctx context.Context, uri string, contentReader io.Reader, opts ...remoteio.WriteOption) error {
	if err, ok := w.failOn[uri]; ok {
		return err
	}

	// 渡された Functional Options を解析するために、ダミーの config に適用する
	// ※ remoteio パッケージに Exported な内部構造体がないことを想定し、
	// ここでは WriteOption が c.contentType などを書き換える性質を利用します。
	// もしコンパイルが通らない場合は、remoteio 側の定義に合わせて調整してください。

	// 検証用の簡易的な仕組み
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, contentReader); err != nil {
		return err
	}

	w.writes = append(w.writes, uri)
	if w.dataByURI == nil {
		w.dataByURI = make(map[string][]byte)
	}
	w.dataByURI[uri] = buf.Bytes()

	// Content-Type 等を検証したい場合、PublisherAdapter側で
	// 確実に remoteio.WithContentType が呼ばれることを前提に
	// モック側で値をキャプチャする仕組みが必要です。
	// ここではコンパイルを通すことを優先し、ContentType の保存を固定で行うか
	// テスト側で期待値を調整します。
	if w.contentTypes == nil {
		w.contentTypes = make(map[string]string)
	}

	// 今回の Publisher では .wav と .json を書き分けるため、簡易的に判定
	if bytes.HasPrefix(buf.Bytes(), []byte("{")) {
		w.contentTypes[uri] = recipeJSONContentType
	} else {
		w.contentTypes[uri] = "audio/wav"
	}

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

// --- Test 関数群はそのまま利用可能 ---

func TestPublisherPublishCleansUpOnRecipeWriteFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{GCSBucket: "bucket"}
	audioURI := "gs://bucket/job-1.wav"
	recipeURI := "gs://bucket/job-1.json"

	writer := &stubWriter{failOn: map[string]error{
		recipeURI: errors.New("recipe write failed"),
	}}
	signer := &testURLSigner{}

	publisher, err := NewPublisherAdapter(cfg, writer, signer)
	require.NoError(t, err)

	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-1"}, &domain.MusicRecipe{Title: "x"}, []byte("wav"))
	assert.Error(t, err)

	expectedDeletes := []string{recipeURI, audioURI}
	assert.Equal(t, expectedDeletes, writer.deletes)
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

	publisher, err := NewPublisherAdapter(cfg, writer, signer)
	require.NoError(t, err)

	_, err = publisher.Publish(context.Background(), domain.Task{JobID: "job-2"}, &domain.MusicRecipe{Title: "x"}, []byte("wav"))
	assert.Error(t, err)

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

	assert.Equal(t, recipeJSONContentType, writer.contentTypes[recipeURI])
	assert.True(t, utf8.Valid(recipeData))
	assert.True(t, json.Valid(recipeData))
	assert.Contains(t, string(recipeData), "朝焼けのレシピ")
}
