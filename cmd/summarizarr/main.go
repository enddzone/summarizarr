package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"summarizarr/internal/ai"
	"summarizarr/internal/api"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"summarizarr/internal/ollama"
	signalclient "summarizarr/internal/signal"
	"time"
)

func main() {
	cfg := config.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	slog.Info("Summarizarr starting...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Ensure the database directory exists
	dbDir := filepath.Dir(cfg.DatabasePath)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			slog.Error("Failed to create database directory", "error", err, "path", dbDir)
			os.Exit(1)
		}
	}

	// Ensure the models directory exists
	if err := os.MkdirAll(cfg.ModelsPath, 0755); err != nil {
		slog.Error("Failed to create models directory", "error", err, "path", cfg.ModelsPath)
		os.Exit(1)
	}

	db, err := database.NewDB(cfg.DatabasePath)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Initialize Ollama manager
	ollamaManager := ollama.NewManager(cfg.ModelsPath, cfg.OllamaHost)

	// Start Ollama server if auto-download is enabled
	if cfg.OllamaAutoDownload {
		slog.Info("Ensuring Ollama is installed...")
		if err := ollamaManager.EnsureInstalled(ctx); err != nil {
			slog.Error("Failed to install Ollama", "error", err)
			os.Exit(1)
		}

		slog.Info("Starting Ollama server...")
		if err := ollamaManager.Start(ctx); err != nil {
			slog.Error("Failed to start Ollama server", "error", err)
			os.Exit(1)
		}

		// Ensure Ollama is stopped on shutdown
		defer func() {
			if err := ollamaManager.Stop(); err != nil {
				slog.Error("Failed to stop Ollama server", "error", err)
			}
		}()
	}

	// Create AI client (now uses local Ollama)
	aiClient, err := ai.NewClient(cfg, db)
	if err != nil {
		slog.Error("Failed to create AI client", "error", err)
		os.Exit(1)
	}

	// Ensure model is downloaded and test the AI client
	if cfg.OllamaAutoDownload {
		if err := ensureModelAndTest(ctx, aiClient, cfg, db); err != nil {
			slog.Error("Failed to ensure model and test AI client", "error", err)
			os.Exit(1)
		}
	}

	apiServer := api.NewServer(":8081", db.DB)

	go apiServer.Start()

	if cfg.PhoneNumber == "" {
		slog.Error("SIGNAL_PHONE_NUMBER environment variable is required")
		os.Exit(1)
	}

	// Use phone number from config
	client := signalclient.NewClient("signal-cli-rest-api:8080", cfg.PhoneNumber, db)

	go func() {
		if err := client.Listen(ctx); err != nil {
			slog.Error("Signal listener error", "error", err)
			os.Exit(1)
		}
	}()

	// Parse summarization interval from config
	summarizationInterval, err := time.ParseDuration(cfg.SummarizationInterval)
	if err != nil {
		slog.Error("Invalid summarization interval", "error", err, "interval", cfg.SummarizationInterval)
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

// ensureModelAndTest downloads the model if needed and tests the AI functionality
func ensureModelAndTest(ctx context.Context, aiClient *ai.Client, cfg *config.Config, db *database.DB) error {
	// Get the backend client to access Ollama-specific methods
	backend, ok := aiClient.GetBackend().(*ollama.Client)
	if !ok {
		return fmt.Errorf("expected Ollama backend, got different type")
	}

	// First, check if Ollama server is healthy
	slog.Info("Checking Ollama server health...")
	if err := backend.HealthCheck(ctx); err != nil {
		return fmt.Errorf("ollama health check failed: %w", err)
	}
	slog.Info("Ollama server is healthy")

	// Ensure the model is downloaded
	slog.Info("Ensuring model is available", "model", cfg.LocalModel)
	if err := backend.EnsureModel(ctx, true); err != nil {
		return fmt.Errorf("failed to ensure model: %w", err)
	}

	// Warm up the model to load it into memory
	slog.Info("Warming up model (this may take a moment for first load)...")
	if err := backend.WarmupModel(ctx); err != nil {
		return fmt.Errorf("failed to warm up model: %w", err)
	}

	// Test with a simple prompt
	slog.Info("Testing model with simple prompt...")
	testMessages := []database.MessageForSummary{
		{
			UserName:    "Alice",
			Text:        "Hello!",
			MessageType: "regular",
		},
		{
			UserName:    "Bob",
			Text:        "Hi there!",
			MessageType: "regular",
		},
	}

	// Use a fresh context with extended timeout for test
	testCtx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	// Add progress logging for the test
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-testCtx.Done():
				return
			case <-ticker.C:
				slog.Info("Still waiting for test summarization to complete...")
			}
		}
	}()

	summary, err := aiClient.Summarize(testCtx, testMessages)
	if err != nil {
		return fmt.Errorf("test summarization failed: %w", err)
	}

	slog.Info("Model test successful", "summary", summary)

	// Test with actual messages from database if available
	groupIDs, err := db.GetGroups()
	if err != nil {
		slog.Warn("Could not get groups for real message test", "error", err)
		return nil // Don't fail on this, test prompt worked
	}

	if len(groupIDs) > 0 {
		// Get recent messages from the first group
		groupID := groupIDs[0]
		now := time.Now().Unix()
		oneDayAgo := now - (24 * 60 * 60) // 24 hours ago

		messages, err := db.GetMessagesForSummarization(groupID, oneDayAgo, now)
		if err != nil {
			slog.Warn("Could not get real messages for test", "error", err)
			return nil
		}

		if len(messages) > 0 {
			slog.Info("Testing with real messages from database", "group_id", groupID, "message_count", len(messages))

			// Use fresh context for real message test too
			realTestCtx, realCancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer realCancel()

			realSummary, err := aiClient.Summarize(realTestCtx, messages)
			if err != nil {
				slog.Warn("Real message summarization test failed", "error", err)
				return nil // Don't fail, basic test worked
			}
			slog.Info("Real message summarization test successful", "summary", realSummary)
		} else {
			slog.Info("No recent messages found for real message test")
		}
	} else {
		slog.Info("No groups found for real message test")
	}

	return nil
}
