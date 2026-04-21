package builder

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/shouni/gcp-kit/tasks"

	"ap-music/internal/config"
	"ap-music/internal/domain"
)

// buildTaskEnqueuer は、Cloud Tasks エンキューアを初期化します。
func buildTaskEnqueuer(ctx context.Context, cfg *config.Config) (*tasks.Enqueuer[domain.Task], error) {
	workerURL, err := url.JoinPath(cfg.ServiceURL, "/tasks/generate")
	if err != nil {
		return nil, fmt.Errorf("failed to build worker URL: %w", err)
	}

	slog.Info("DEBUG: Building Task Enqueuer",
		"project_id", cfg.ProjectID,
		"location_id", cfg.LocationID,
		"queue_id", cfg.QueueID,
		"worker_url", workerURL,
		"sa_email", cfg.ServiceAccountEmail,
	)

	taskCfg := tasks.Config{
		ProjectID:           cfg.ProjectID,
		LocationID:          cfg.LocationID,
		QueueID:             cfg.QueueID,
		WorkerURL:           workerURL,
		ServiceAccountEmail: cfg.ServiceAccountEmail,
		Audience:            cfg.TaskAudienceURL,
	}

	// リソースパスのプレビューも出すとさらに確実です
	parentPath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", cfg.ProjectID, cfg.LocationID, cfg.QueueID)
	slog.Info("DEBUG: Generated Parent Path", "path", parentPath)

	return tasks.NewEnqueuer[domain.Task](ctx, taskCfg)
}
