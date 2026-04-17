package server

import (
	"net/http"

	"ap-music/internal/controllers/web"
	"ap-music/internal/controllers/worker"
)

// NewRouter は HTTP ルーティングを組み立てます。
func NewRouter(webHandler web.Handler, workerHandler worker.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /web/compose", webHandler.EnqueueTask)
	mux.HandleFunc("POST /worker/tasks", workerHandler.ExecuteTask)
	return mux
}
