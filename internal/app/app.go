package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"ap-music/internal/builder"
	"ap-music/internal/config"
)

// Run はアプリケーションを起動します。
func Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: builder.BuildRouter(cfg),
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-sigCh:
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
