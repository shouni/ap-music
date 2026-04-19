package adapters

import (
	"ap-music/internal/config"
	"bytes"
	"context"
	"fmt"

	"ap-music/internal/domain"

	"github.com/shouni/go-remote-io/remoteio"
)

// PublisherAdapter は成果物保存を行う雛形です。
type PublisherAdapter struct {
	writer remoteio.Writer
	signer remoteio.URLSigner
	Bucket string
}

// NesPublisherAdapter は成果物保存のためのアダプターを生成します。
func NesPublisherAdapter(cfg *config.Config, writer remoteio.Writer, signer remoteio.URLSigner) (*PublisherAdapter, error) {
	return &PublisherAdapter{
		writer: writer,
		signer: signer,
		Bucket: cfg.GCSBucket,
	}, nil
}

// Publish は保存結果を返します。
// Publish は保存結果を返します。
func (a PublisherAdapter) Publish(ctx context.Context, task domain.Task, outputFile []byte) (domain.PublishResult, error) {
	if task.JobID == "" {
		return domain.PublishResult{}, fmt.Errorf("job id is required")
	}

	storageURI := fmt.Sprintf("gs://%s/%s.wav", a.Bucket, task.JobID)

	// 署名付きURLの生成
	signedURL, err := a.generateSignedResultURL(ctx, storageURI)
	if err != nil {
		return domain.PublishResult{}, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	// []byte を io.Reader に変換して Writer を呼び出し
	contentReader := bytes.NewReader(outputFile)
	if err := a.writer.Write(ctx, storageURI, contentReader, "audio/wav"); err != nil {
		return domain.PublishResult{}, fmt.Errorf("failed to write to storage: %w", err)
	}

	return domain.PublishResult{
		JobID:      task.JobID,
		StorageURI: storageURI,
		SignedURL:  signedURL,
	}, nil
}

// generateSignedResultURL は StorageURI から署名付きURLを作るヘルパーです。
func (a PublisherAdapter) generateSignedResultURL(ctx context.Context, storageURI string) (string, error) {
	return a.signer.GenerateSignedURL(ctx, storageURI, "GET", config.SignedURLExpiration)
}
