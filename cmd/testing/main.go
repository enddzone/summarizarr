package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"summarizarr/internal/ai"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"time"
)

func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: testing <backend>")
		fmt.Println("  backend: 'local' or 'openai'")
		fmt.Println("")
		fmt.Println("Environment variables required:")
		fmt.Println("  For local backend: LOCAL_MODEL, OLLAMA_HOST (optional)")
		fmt.Println("  For OpenAI backend: OPENAI_API_KEY, OPENAI_MODEL (optional)")
		os.Exit(1)
	}

	backend := os.Args[1]
	if backend != "local" && backend != "openai" {
		fmt.Printf("Error: unsupported backend '%s'. Use 'local' or 'openai'\n", backend)
		os.Exit(1)
	}

	// Override AI_BACKEND for testing
	os.Setenv("AI_BACKEND", backend)

	// Create configuration
	cfg := config.New()

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	fmt.Printf("Testing %s backend...\n", backend)

	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		fmt.Printf("Configuration error: %s\n", err)
		os.Exit(1)
	}

	// Create AI client
	aiClient, err := ai.NewClient(cfg, nil) // No database for simple test
	if err != nil {
		fmt.Printf("Failed to create AI client: %s\n", err)
		os.Exit(1)
	}

	// Test with simple messages
	testMessages := []database.MessageForSummary{
		{
			UserID:      1,
			UserName:    "Alice",
			Text:        "Hello everyone! How is your day going?",
			MessageType: "regular",
		},
		{
			UserID:      2,
			UserName:    "Bob",
			Text:        "Hi Alice! My day is going great, thanks for asking. Just finished a big project at work.",
			MessageType: "regular",
		},
		{
			UserID:      1,
			UserName:    "Alice",
			Text:        "That's awesome Bob! What kind of project was it?",
			MessageType: "regular",
		},
		{
			UserID:      2,
			UserName:    "Bob",
			Text:        "It was a machine learning model for predicting customer behavior. Really exciting stuff!",
			MessageType: "regular",
		},
	}

	fmt.Printf("Testing summarization with %d test messages...\n", len(testMessages))

	// Test summarization
	testCtx, testCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer testCancel()

	summary, err := aiClient.Summarize(testCtx, testMessages)
	if err != nil {
		fmt.Printf("Summarization failed: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ %s backend test successful!\n", backend)
	fmt.Printf("Summary length: %d characters\n", len(summary))
	fmt.Printf("Summary: %s\n", summary)
}

// validateConfig validates backend-specific configuration requirements
func validateConfig(cfg *config.Config) error {
	// Check required environment variables based on backend
	switch cfg.AIBackend {
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return fmt.Errorf("OPENAI_API_KEY is required when AI_BACKEND=openai")
		}
		if cfg.OpenAIModel == "" {
			return fmt.Errorf("OPENAI_MODEL is required when AI_BACKEND=openai")
		}
		fmt.Printf("Using OpenAI backend with model: %s\n", cfg.OpenAIModel)
	case "local":
		if cfg.LocalModel == "" {
			return fmt.Errorf("LOCAL_MODEL is required when AI_BACKEND=local")
		}
		fmt.Printf("Using Ollama backend with model: %s (host: %s)\n", cfg.LocalModel, cfg.OllamaHost)
	default:
		return fmt.Errorf("unsupported AI_BACKEND: %s (supported: 'local', 'openai')", cfg.AIBackend)
	}

	return nil
}
