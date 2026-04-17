package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"ap-music/internal/config"
	"ap-music/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to load config: %w", err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, &cfg); err != nil {
		log.Fatal(err)
	}
}
