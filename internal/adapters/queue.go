package adapters

import (
	"context"
	"fmt"

	"ap-music/internal/domain"
)

// CloudTasksAdapter は Cloud Tasks 投入の雛形です。
type CloudTasksAdapter struct{}

// Enqueue はジョブを投入します。
func (CloudTasksAdapter) Enqueue(_ context.Context, task domain.Task) error {
	if task.JobID == "" {
		return fmt.Errorf("job id is required")
	}
	return nil
}
