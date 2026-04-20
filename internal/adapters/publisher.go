package adapters

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/shouni/go-remote-io/remoteio"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// PublisherAdapter は成果物保存を行うアダプターです。
type PublisherAdapter struct {
	writer     remoteio.Writer
	signer     remoteio.URLSigner
	Bucket     string
	Expiration time.Duration
}

// NewPublisherAdapter は成果物保存のためのアダプターを生成します。
func NewPublisherAdapter(cfg *config.Config, writer remoteio.Writer, signer remoteio.URLSigner) (*PublisherAdapter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	return &PublisherAdapter{
		writer:     writer,
		signer:     signer,
		Bucket:     cfg.GCSBucket,
		Expiration: config.SignedURLExpiration,
	}, nil
}

// Publish は成果物をストレージに保存し、その結果（署名付きURL等）を返します。
func (a *PublisherAdapter) Publish(ctx context.Context, task domain.Task, audioData []byte) (*domain.PublishResult, error) {
	if task.JobID == "" {
		return nil, fmt.Errorf("job id is required")
	}
	if len(audioData) == 0 {
		return nil, fmt.Errorf("output file is empty")
	}

	storageURI := remoteio.BuildGCSURI(a.Bucket, fmt.Sprintf("%s.wav", task.JobID))
	signedURL, err := a.generateSignedResultURL(ctx, storageURI)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signed URL: %w", err)
	}

	contentReader := bytes.NewReader(audioData)
	if err := a.writer.Write(ctx, storageURI, contentReader, "audio/wav"); err != nil {
		return nil, fmt.Errorf("failed to write to storage: %w", err)
	}

	return &domain.PublishResult{
		JobID:      task.JobID,
		StorageURI: storageURI,
		SignedURL:  signedURL,
	}, nil
}

// generateSignedResultURL は StorageURI から署名付きURLを作るヘルパーです。
func (a *PublisherAdapter) generateSignedResultURL(ctx context.Context, storageURI string) (string, error) {
	return a.signer.GenerateSignedURL(ctx, storageURI, "GET", a.Expiration)
}
