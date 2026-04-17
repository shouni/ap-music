package builder

import (
	"net/http"

	"ap-music/internal/adapters"
	"ap-music/internal/config"
	"ap-music/internal/controllers/web"
	"ap-music/internal/controllers/worker"
	"ap-music/internal/pipeline"
	"ap-music/internal/server"
)

// BuildRouter は依存関係を組み立て、HTTPルータを返します。
func BuildRouter(cfg config.Config) http.Handler {
	reader := adapters.ReaderAdapter{}
	lyria := adapters.LyriaAdapter{Model: cfg.LyriaModel}
	publisher := adapters.PublisherAdapter{Bucket: cfg.GCSBucket}
	notifier := adapters.SlackAdapter{WebhookURL: cfg.SlackWebhookURL}
	queue := adapters.CloudTasksAdapter{}

	wf := pipeline.MusicPipeline{
		Collector: reader,
		Composer:  lyria,
		Generator: lyria,
		Publisher: publisher,
		Notifier:  notifier,
	}

	webHandler := web.Handler{Queue: queue}
	workerHandler := worker.Handler{Workflow: wf}
	return server.NewRouter(webHandler, workerHandler)
}
