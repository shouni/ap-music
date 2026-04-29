package builder

import (
	"context"
	"fmt"
	"io"

	gcsstorage "cloud.google.com/go/storage"
	"github.com/shouni/go-http-kit/httpkit"
	"github.com/shouni/go-remote-io/remoteio/gcs"

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
	storage, err := gcs.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS factory: %w", err)
	}
	resources = append(resources, storage)

	cleanupClient, err := gcsstorage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS cleanup client: %w", err)
	}
	resources = append(resources, cleanupClient)

	rio, err := buildRemoteIO(storage)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize IO components: %w", err)
	}

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

	aiClient, err := adapters.NewVertexAIAdapter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AI adapter: %w", err)
	}

	// 3. Prompt Generator
	promptGen, err := adapters.NewPromptAdapter()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prompt adapter: %w", err)
	}

	// 4. Music Runner
	runner, err := adapters.NewLyriaAdapter(ctx, cfg, aiClient, promptGen)
	if err != nil {
		return nil, err
	}

	reader, err := adapters.NewReaderAdapter(rio.Factory)
	if err != nil {
		return nil, err
	}

	publisher, err := adapters.NewPublisherAdapter(
		cfg,
		rio.Writer,
		rio.Signer,
		adapters.NewStorageCleaner(cleanupClient),
	)
	if err != nil {
		return nil, err
	}

	// 5. Pipeline (Core Logic)
	pipeline, err := buildPipeline(reader, runner, publisher, slack)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize music pipeline: %w", err)
	}

	appCtx := &app.Container{
		Config:       cfg,
		RemoteIO:     rio,
		TaskEnqueuer: enqueuer,
		Pipeline:     pipeline,
		HTTPClient:   httpClient,
		Notifier:     slack,
	}

	return appCtx, nil
}
