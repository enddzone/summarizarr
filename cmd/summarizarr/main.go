package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"summarizarr/internal/ai"
	"summarizarr/internal/api"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"time"
	signalclient "summarizarr/internal/signal"
)

func main() {
	cfg := config.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	slog.Info("Summarizarr starting...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	db, err := database.NewDB("summarizarr.db")
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	aiModel := os.Getenv("OPENAI_MODEL")
	if aiModel == "" {
		aiModel = "gpt-4o"
	}

	aiClient := ai.NewClient(os.Getenv("OPENAI_API_KEY"), aiModel)
	apiServer := api.NewServer(":8081", db.DB)

	go apiServer.Start()

	// Replace with your actual address and number
	client := signalclient.NewClient("signal-cli-rest-api:8080", "+18177392137", db)

	go func() {
		if err := client.Listen(ctx); err != nil {
			slog.Error("Signal listener error", "error", err)
			os.Exit(1)
		}
	}()

	summarizationIntervalStr := os.Getenv("SUMMARIZATION_INTERVAL")
	if summarizationIntervalStr == "" {
		summarizationIntervalStr = "12h"
	}

	summarizationInterval, err := time.ParseDuration(summarizationIntervalStr)
	if err != nil {
		slog.Error("Invalid summarization interval", "error", err)
		os.Exit(1)
	}

	scheduler := ai.NewScheduler(db, aiClient, summarizationInterval)
	go scheduler.Start(ctx)

	<-ctx.Done()
	slog.Info("Shutting down Summarizarr...")
	if err := apiServer.Shutdown(ctx); err != nil {
		slog.Error("API server shutdown error", "error", err)
	}
}