package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"ap-music/internal/builder"
	"ap-music/internal/config"
)

// NewRouter は、ミドルウェアとルーティングを統合した http.Handler を構築します。
func NewRouter(cfg *config.Config, h *builder.AppHandlers) http.Handler {
	r := chi.NewRouter()
	setupCommonMiddleware(r)
	setupRoutes(r, cfg, h)

	return r
}

// setupCommonMiddleware は、標準的なミドルウェアを構成します。
func setupCommonMiddleware(r *chi.Mux) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
}

// setupRoutes は、各コンポーネントのハンドラーをルーティングに登録します。
func setupRoutes(r chi.Router, cfg *config.Config, h *builder.AppHandlers) {
	// --- 1. 公開ルート (ヘルスチェック) ---
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	if h == nil {
		slog.Warn("AppHandlers is nil, skipping application routes registration")
		return
	}

	// --- 2. 認証関連エンドポイント (OAuth2 フロー) ---
	if h.Auth != nil {
		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", h.Auth.Login)
			r.Get("/callback", h.Auth.Callback)
		})
	}

	// --- 3. 認証が必要なルート (Web UI 用) ---
	r.Group(func(r chi.Router) {
		if h.Auth == nil {
			if h.Web != nil {
				slog.Error("Auth handler is nil, skipping protected web routes")
			}
			return
		}

		// ログインチェック & POST時のCSRF検証を適用
		r.Use(h.Auth.Middleware)

		// GETリクエスト時にCSRFトークンがなければ自動生成してセッションに保存するミドルウェア
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					// セッションにトークンがない場合のみ生成
					if h.Auth.GetCSRFTokenFromSession(r) == "" {
						if _, err := h.Auth.GenerateAndSaveCSRFToken(w, r); err != nil {
							slog.Error("Failed to auto-generate CSRF token", "error", err)
							http.Error(w, "Internal Server Error", http.StatusInternalServerError)
							return
						}
					}
				}
				next.ServeHTTP(w, r)
			})
		})

		if h.Web != nil {
			r.Get("/", h.Web.Home)
			r.Post("/web/compose", h.Web.EnqueueTask)
			r.Route("/web/history", func(r chi.Router) {
				r.Get("/", h.Web.ServeHistory)
				r.Get("/{jobID}", h.Web.ServeDetails)
				r.Delete("/{jobID}", h.Web.DeleteHistory)
			})
			r.Get("/web/audio/{jobID}", h.Web.ServeAudio)
		}
	})

	// --- 4. Cloud Tasks 専用ルート (Worker 用) ---
	r.Group(func(r chi.Router) {
		if h.Auth == nil {
			if h.Worker != nil {
				slog.Error("Auth handler is nil, skipping worker routes")
			}
			return
		}

		// Cloud Tasks からの OIDC トークンを検証 (セッション不要)
		r.Use(h.Auth.TaskOIDCVerificationMiddleware)

		if h.Worker != nil {
			r.Post("/tasks/generate", h.Worker.ProcessTask)
		}
	})
}
