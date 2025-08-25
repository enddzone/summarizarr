package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"summarizarr/internal/ai"
	"summarizarr/internal/api"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"summarizarr/internal/encryption"
	"summarizarr/internal/frontend"
	"summarizarr/internal/ollama"
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

	// Preflight: If an existing DB is plaintext (unencrypted), back it up and allow creation of a fresh encrypted DB
	if err := backupIfPlaintext(cfg.DatabasePath); err != nil {
		slog.Error("Preflight encryption check failed", "error", err)
		os.Exit(1)
	}

	// Mandatory encryption: load or create key using manager
	encMgr := encryption.NewManager(cfg.DatabasePath)
	encKey, err := encMgr.LoadOrCreateKey()
	if err != nil {
		slog.Error("Failed to obtain encryption key", "error", err)
		os.Exit(1)
	}

	db, err := database.NewDB(cfg.DatabasePath, encKey)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close database", "error", err)
		}
	}()

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Initialize AI backend based on configuration (switch improves readability, satisfies staticcheck suggestion)
	switch cfg.AIProvider {
	case "local":
		slog.Info("AI_PROVIDER=local detected. Using external Ollama server...")
	case "openai":
		slog.Info("AI_PROVIDER=openai detected. Initializing OpenAI backend...")
		if cfg.OpenAIAPIKey == "" { // Validate required OpenAI configuration
			slog.Error("OPENAI_API_KEY environment variable is required when AI_PROVIDER=openai")
			os.Exit(1)
		}
	case "groq", "gemini", "claude":
		// These are validated later in validateConfig/testAIProvider
		// Log detection for observability
		slog.Info("Detected AI provider", "provider", cfg.AIProvider)
	default:
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
	apiServer := api.NewServerWithAppDB(cfg.ListenAddr, db, frontendFS)

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

	// Rotation scheduler removed

	<-ctx.Done()
	slog.Info("Shutting down Summarizarr...")
	if err := apiServer.Shutdown(ctx); err != nil {
		slog.Error("API server shutdown error", "error", err)
	}
}

// backupIfPlaintext inspects the first 16 bytes of the DB file. If it matches the
// plain SQLite header ("SQLite format 3\x00"), it moves the file (and sidecars) to a timestamped backup
// so the application can create a new encrypted database on first run.
func backupIfPlaintext(dbPath string) error {
	if dbPath == "" || dbPath == ":memory:" {
		return nil
	}

	// Validate path early to avoid surprising errors later
	if err := validateDatabasePath(dbPath); err != nil {
		return err
	}
    f, err := os.Open(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to do
		}
		return fmt.Errorf("failed to open database file: %w", err)
	}
    defer func() { _ = f.Close() }()
	hdr := make([]byte, 16)
	n, err := f.Read(hdr)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("failed to read database header: %w", err)
	}
	if n < 16 {
		return nil // small or empty file; let normal init proceed
	}
	if string(hdr) == "SQLite format 3\x00" {
		ts := time.Now().Format("20060102_150405")
		backupPath := filepath.Join(filepath.Dir(dbPath), fmt.Sprintf("%s_backup_%s.db", filepath.Base(dbPath), ts))
		if err := os.Rename(dbPath, backupPath); err != nil {
			return fmt.Errorf("failed to back up plaintext DB: %w", err)
		}
		// Best-effort move sidecar files too
		_ = moveIfExists(dbPath+"-wal", backupPath+"-wal")
		_ = moveIfExists(dbPath+"-shm", backupPath+"-shm")
		slog.Warn("Detected plaintext SQLite DB; moved to backup to create encrypted DB", "from", dbPath, "to", backupPath)
	}
	return nil
}

func moveIfExists(src, dst string) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Rename(src, dst)
}

// validateDatabasePath ensures dbPath is a file path (not a dir), resolves absolute path,
// and that its parent directory exists and is accessible. It allows non-existent files.
func validateDatabasePath(dbPath string) error {
	if dbPath == "" || dbPath == ":memory:" {
		return nil
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return fmt.Errorf("invalid database path: %w", err)
	}
	// Ensure parent dir exists (create earlier in main, but validate here too)
	dir := filepath.Dir(abs)
	if dir == "." || dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("database directory does not exist: %s", dir)
		}
		return fmt.Errorf("failed to stat database directory: %w", err)
	}
	// If a path exists and is directory, reject
	if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
		return fmt.Errorf("database path is a directory: %s", abs)
	}
	return nil
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

// testOllamaBackend tests the Ollama backend using external sidecar with comprehensive validation
func testOllamaBackend(ctx context.Context, aiClient *ai.Client, cfg *config.Config, db *database.DB) error {
	slog.Info("Performing comprehensive validation of external Ollama server...")

	// Get the Ollama client from the AI client for direct validation
	if err := validateOllamaStartup(ctx, cfg); err != nil {
		return fmt.Errorf("ollama startup validation failed: %w", err)
	}

	// Perform end-to-end test through AI client
	slog.Info("Testing end-to-end summarization through AI client...")
	testMessages := []database.MessageForSummary{
		{
			UserName:    "TestUser",
			Text:        "Hello, testing connectivity!",
			MessageType: "regular",
		},
	}

	// Use a context with reasonable timeout for the full test
	testCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	summary, err := aiClient.Summarize(testCtx, testMessages)
	if err != nil {
		slog.Warn("End-to-end test failed, but continuing startup", "error", err)
		return nil // Graceful degradation - log warning but don't fail startup
	}

	if summary == "" {
		slog.Warn("End-to-end test returned empty response, but continuing startup")
		return nil // Graceful degradation
	}

	slog.Info("External Ollama backend fully validated", "model", cfg.LocalModel, "host", cfg.OllamaHost, "summary_length", len(summary))
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

// validateOllamaStartup performs comprehensive validation of the external Ollama server with retry logic
func validateOllamaStartup(ctx context.Context, cfg *config.Config) error {
	client := ollama.NewClient(cfg.OllamaHost, cfg.LocalModel)

	// Implement retry logic for server connectivity
	maxRetries := 5
	retryInterval := 2 * time.Second

	slog.Info("Validating Ollama server startup", "host", cfg.OllamaHost, "model", cfg.LocalModel, "max_retries", maxRetries)

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create a timeout context for each validation attempt
		validateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

		// Attempt comprehensive validation
		err := client.ValidateExternalOllama(validateCtx)
		cancel()

		if err == nil {
			slog.Info("Ollama startup validation successful", "attempt", attempt, "total_attempts", maxRetries)
			return nil
		}

		lastErr = err

		if attempt < maxRetries {
			slog.Warn("Ollama validation failed, retrying...",
				"attempt", attempt,
				"max_retries", maxRetries,
				"retry_in_seconds", int(retryInterval.Seconds()),
				"error", err)

			// Wait before retrying, but respect context cancellation
			select {
			case <-ctx.Done():
				return fmt.Errorf("validation cancelled: %w", ctx.Err())
			case <-time.After(retryInterval):
				// Exponential backoff with jitter
				retryInterval = time.Duration(float64(retryInterval) * 1.5)
				if retryInterval > 10*time.Second {
					retryInterval = 10 * time.Second
				}
			}
		}
	}

	// If we get here, all attempts failed
	return fmt.Errorf("ollama validation failed after %d attempts. Last error: %w\n\nCommon solutions:\n1. Start Ollama: 'ollama serve'\n2. Pull the model: 'ollama pull %s'\n3. Check Ollama is running on the correct host: %s",
		maxRetries, lastErr, cfg.LocalModel, cfg.OllamaHost)
}
