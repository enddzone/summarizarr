package ai

import (
	"context"
	"fmt"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"summarizarr/internal/ollama"
)

// AIClient defines the interface for AI summarization services.
type AIClient interface {
	Summarize(ctx context.Context, messages []database.MessageForSummary) (string, error)
}

// Client wraps an AI backend client.
type Client struct {
	backend AIClient
}

// NewClient creates a new AI client based on the configuration.
func NewClient(cfg *config.Config) (*Client, error) {
	var backend AIClient

	switch cfg.AIBackend {
	case "local":
		// Create Ollama client
		backend = ollama.NewClient(cfg.OllamaHost, cfg.LocalModel)
	default:
		return nil, fmt.Errorf("unsupported AI backend: %s (only 'local' is supported)", cfg.AIBackend)
	}

	return &Client{backend: backend}, nil
}

// Summarize delegates to the backend implementation.
func (c *Client) Summarize(ctx context.Context, messages []database.MessageForSummary) (string, error) {
	return c.backend.Summarize(ctx, messages)
}

// GetBackend returns the underlying AI backend for type-specific operations
func (c *Client) GetBackend() AIClient {
	return c.backend
}
