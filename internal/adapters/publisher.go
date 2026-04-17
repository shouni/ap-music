package adapters

import (
	"context"
	"fmt"

	"ap-music/internal/domain"
)

// PublisherAdapter は成果物保存を行う雛形です。
type PublisherAdapter struct {
	Bucket string
}

// Publish は保存結果を返します。
func (a PublisherAdapter) Publish(_ context.Context, task domain.Task, _ []byte) (domain.PublishResult, error) {
	if task.JobID == "" {
		return domain.PublishResult{}, fmt.Errorf("job id is required")
	}
	return domain.PublishResult{
		JobID:      task.JobID,
		StorageURI: fmt.Sprintf("gs://%s/%s.mp3", a.Bucket, task.JobID),
		SignedURL:  fmt.Sprintf("https://example.com/music/%s", task.JobID),
	}, nil
}
