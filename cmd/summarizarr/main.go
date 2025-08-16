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
	"summarizarr/internal/frontend"
	signalclient "summarizarr/internal/signal"
	"summarizarr/internal/version"
	"time"
)

func main() {
	cfg := config.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	slog.Info("Summarizarr starting...", "version", version.GetVersion())

	// Validate backend-specific configuration
	if err := validateConfig(cfg); err != nil {
		slog.Error("Configuration validation failed", "error", err)
		os.Exit(1)
	}

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

	// Initialize AI backend based on configuration
	if cfg.AIProvider == "local" {
		slog.Info("AI_PROVIDER=local detected. Using external Ollama server...")
	} else if cfg.AIProvider == "openai" {
		slog.Info("AI_PROVIDER=openai detected. Initializing OpenAI backend...")

		// Validate required OpenAI configuration
		if cfg.OpenAIAPIKey == "" {
			slog.Error("OPENAI_API_KEY environment variable is required when AI_PROVIDER=openai")
			os.Exit(1)
		}
	} else {
		slog.Error("Invalid AI_PROVIDER configuration", "provider", cfg.AIProvider, "supported", "local, openai, groq, gemini, claude")
		os.Exit(1)
	}

	// Create AI client
	aiClient, err := ai.NewClient(cfg, db)
	if err != nil {
		slog.Error("Failed to create AI client", "error", err)
		os.Exit(1)
	}

	// Test the AI backend
	if err := testAIProvider(ctx, aiClient, cfg, db); err != nil {
		slog.Error("Failed to test AI backend", "error", err)
		os.Exit(1)
	}

	// Prepare frontend filesystem
	frontendFS, err := frontend.GetFS()
	if err != nil {
		slog.Error("Failed to get frontend filesystem", "error", err)
		frontendFS = nil
	}

	// API server listen address is configurable via LISTEN_ADDR (default :8080)
	apiServer := api.NewServer(cfg.ListenAddr, db.DB, frontendFS)

	go apiServer.Start()

	if cfg.PhoneNumber == "" {
		slog.Error("SIGNAL_PHONE_NUMBER environment variable is required")
		os.Exit(1)
	}

	// Use phone number and Signal URL from config
	client := signalclient.NewClient(cfg.SignalURL, cfg.PhoneNumber, db)

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

// testAIProvider tests the AI provider and ensures it's ready
func testAIProvider(ctx context.Context, aiClient *ai.Client, cfg *config.Config, db *database.DB) error {
	switch cfg.AIProvider {
	case "local":
		return testOllamaBackend(ctx, aiClient, cfg, db)
	case "openai":
		return testOpenAIBackend(ctx, aiClient, cfg)
	case "groq", "gemini", "claude":
		// These providers use the same client as OpenAI, so we can test them the same way
		return testOpenAIBackend(ctx, aiClient, cfg)
	default:
		return fmt.Errorf("unsupported AI provider: %s", cfg.AIProvider)
	}
}

// testOllamaBackend tests the Ollama backend using external sidecar
func testOllamaBackend(ctx context.Context, aiClient *ai.Client, cfg *config.Config, db *database.DB) error {
	slog.Info("Testing external Ollama server connectivity...")

	// Create a simple test message
	testMessages := []database.MessageForSummary{
		{
			UserName:    "TestUser",
			Text:        "Hello, testing connectivity!",
			MessageType: "regular",
		},
	}

	// Use a context with reasonable timeout
	testCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	summary, err := aiClient.Summarize(testCtx, testMessages)
	if err != nil {
		return fmt.Errorf("external Ollama backend test failed: %w", err)
	}

	if summary == "" {
		return fmt.Errorf("external Ollama backend returned empty response")
	}

	slog.Info("External Ollama backend is ready", "model", cfg.LocalModel, "host", cfg.OllamaHost)
	return nil
}

// testOpenAIBackend tests the OpenAI backend
func testOpenAIBackend(ctx context.Context, aiClient *ai.Client, cfg *config.Config) error {
	slog.Info("Testing OpenAI backend connectivity...")

	// Create a simple test message
	testMessages := []database.MessageForSummary{
		{
			UserID:      1,
			UserName:    "TestUser",
			Text:        "Hello, are you there?",
			MessageType: "regular",
		},
	}

	// Use a context with reasonable timeout
	testCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	summary, err := aiClient.Summarize(testCtx, testMessages)
	if err != nil {
		return fmt.Errorf("OpenAI backend test failed: %w", err)
	}

	if summary == "" {
		return fmt.Errorf("OpenAI backend returned empty response")
	}

	slog.Info("OpenAI backend is ready", "model", cfg.OpenAIModel, "test_response_length", len(summary))
	return nil
}

// validateConfig validates provider-specific configuration requirements
func validateConfig(cfg *config.Config) error {
	// Check required environment variables based on provider
	switch cfg.AIProvider {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when AI_PROVIDER=openai")
		}
		if cfg.OpenAIModel == "" {
			return fmt.Errorf("OPENAI_MODEL is required when AI_PROVIDER=openai")
		}
		slog.Info("Using OpenAI provider", "model", cfg.OpenAIModel)
	case "groq":
		if cfg.GroqAPIKey == "" {
			return fmt.Errorf("GROQ_API_KEY is required when AI_PROVIDER=groq")
		}
		slog.Info("Using Groq provider", "model", cfg.GroqModel)
	case "gemini":
		if cfg.GeminiAPIKey == "" {
			return fmt.Errorf("GEMINI_API_KEY is required when AI_PROVIDER=gemini")
		}
		slog.Info("Using Gemini provider", "model", cfg.GeminiModel)
	case "claude":
		if cfg.ClaudeAPIKey == "" {
			return fmt.Errorf("CLAUDE_API_KEY is required when AI_PROVIDER=claude")
		}
		slog.Info("Using Claude provider", "model", cfg.ClaudeModel)
	case "local":
		if cfg.LocalModel == "" {
			return fmt.Errorf("LOCAL_MODEL is required when AI_PROVIDER=local")
		}
		slog.Info("Using Ollama provider", "model", cfg.LocalModel, "host", cfg.OllamaHost)
	default:
		return fmt.Errorf("unsupported AI_PROVIDER: %s (supported: 'local', 'openai', 'groq', 'gemini', 'claude')", cfg.AIProvider)
	}

	// Check required general settings
	if cfg.PhoneNumber == "" {
		return fmt.Errorf("SIGNAL_PHONE_NUMBER is required")
	}

	return nil
}
