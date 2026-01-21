package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/altuslabsxyz/alert-bridge/internal/app"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	application, err := app.New(configPath)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Start(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}

	if err := application.Shutdown(); err != nil {
		log.Fatalf("shutdown error: %v", err)
	}
}
