package builder

import (
	"ap-music/internal/pipeline"
	"context"
	"fmt"
	"io"

	"github.com/shouni/go-http-kit/httpkit"

	"ap-music/internal/adapters"
	"ap-music/internal/app"
	"ap-music/internal/config"
)

// BuildContainer は外部サービスとの接続を確立し、依存関係を組み立てた app.Container を返します。
func BuildContainer(ctx context.Context, cfg *config.Config) (container *app.Container, err error) {
	var resources []io.Closer
	defer func() {
		if err != nil {
			for _, r := range resources {
				if r != nil {
					_ = r.Close()
				}
			}
		}
	}()

	// 1. I/O Infrastructure (GCS)
	rio, err := buildRemoteIO(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize IO components: %w", err)
	}
	resources = append(resources, rio)

	// 2. Task Enqueuer
	enqueuer, err := buildTaskEnqueuer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task enqueuer: %w", err)
	}
	resources = append(resources, enqueuer)

	httpClient := httpkit.New(config.DefaultHTTPTimeout)
	slack, err := adapters.NewSlackAdapter(httpClient, cfg.SlackWebhookURL)
	if err != nil {
		return nil, err
	}

	musicAI := adapters.LyriaAdapter(ctx, cfg)

	workflow := pipeline.Workflow()

	// 3. Pipeline (Core Logic)
	mangaPipeline, err := buildPipeline(cfg, workflow, slack)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize manga pipeline: %w", err)
	}

	appCtx := &app.Container{
		Config:       cfg,
		RemoteIO:     rio,
		TaskEnqueuer: enqueuer,
		Pipeline:     mangaPipeline,
		HTTPClient:   httpClient,
		Notifier:     slack,
	}

	return appCtx, nil
}
