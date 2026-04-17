package server

import (
	"net/http"

	"ap-music/internal/builder"
	"ap-music/internal/config"
)

// NewRouter は HTTP ルーティングを組み立てます。
func NewRouter(_ *config.Config, h *builder.AppHandlers) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if h != nil && h.Auth != nil {
		mux.HandleFunc("GET /auth/login", h.Auth.Login)
		mux.HandleFunc("GET /auth/callback", h.Auth.Callback)
	}

	if h != nil && h.Web != nil {
		mux.HandleFunc("GET /", h.Web.Home)
		mux.HandleFunc("POST /web/compose", h.Web.Compose)
	}

	if h != nil && h.Worker != nil {
		workerHandler := http.HandlerFunc(h.Worker.ProcessTask)
		if h.Auth != nil {
			workerHandler = h.Auth.TaskOIDCVerificationMiddleware(workerHandler).ServeHTTP
		}
		mux.Handle("POST /tasks/generate", http.HandlerFunc(workerHandler))
	}

	return mux
}
