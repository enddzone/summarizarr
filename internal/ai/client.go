package ai

import (
	"context"
	"fmt"
	"strings"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"summarizarr/internal/ollama"
	openaiclient "summarizarr/internal/openai"
)

// SummarizationPrompt is the template used for all LLM backends
const SummarizationPrompt = `Please provide a concise summary of this Signal group conversation. Focus on:
- Key topics discussed
- Important decisions or conclusions
- Action items or next steps
- Notable reactions or responses

Conversation format: Regular messages, quoted replies (shown as 'replying to: "original text"'), and emoji reactions.

Conversation:
{{.Messages}}`

// AIClient defines the interface for AI summarization services.
type AIClient interface {
	Summarize(ctx context.Context, prompt string) (string, error)
}

// FormatMessagesForLLM formats messages for LLM consumption, including anonymization
func FormatMessagesForLLM(messages []database.MessageForSummary) string {
	var content strings.Builder

	for _, msg := range messages {
		switch msg.MessageType {
		case "reaction":
			if msg.ReactionEmoji != "" {
				content.WriteString(fmt.Sprintf("user_%d reacted with %s\n", msg.UserID, msg.ReactionEmoji))
			}
		case "quote":
			if msg.QuoteText != "" && msg.Text != "" {
				content.WriteString(fmt.Sprintf("user_%d (replying to: \"%s\"): %s\n", msg.UserID, msg.QuoteText, msg.Text))
			} else if msg.Text != "" {
				content.WriteString(fmt.Sprintf("user_%d: %s\n", msg.UserID, msg.Text))
			}
		default: // regular message
			if msg.Text != "" {
				content.WriteString(fmt.Sprintf("user_%d: %s\n", msg.UserID, msg.Text))
			}
		}
	}

	return content.String()
}

// Client wraps an AI backend client.
type Client struct {
	backend AIClient
	db      DB
}

// NewClient creates a new AI client based on the configuration.
func NewClient(cfg *config.Config, db DB) (*Client, error) {
	var backend AIClient
	switch cfg.AIBackend {
	case "local":
		backend = ollama.NewClient(cfg.OllamaHost, cfg.LocalModel)
	case "openai":
		// Use OpenAI backend - no longer needs db since post-processing is handled here
		backend = openaiclient.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	default:
		return nil, fmt.Errorf("unsupported AI backend: %s (supported: 'local', 'openai')", cfg.AIBackend)
	}
	return &Client{backend: backend, db: db}, nil
}

// Summarize formats messages, creates prompt, calls backend, and handles post-processing
func (c *Client) Summarize(ctx context.Context, messages []database.MessageForSummary) (string, error) {
	// Format messages with anonymization
	formatted := FormatMessagesForLLM(messages)

	// Create prompt using template
	prompt := strings.Replace(SummarizationPrompt, "{{.Messages}}", formatted, 1)

	// Call backend with constructed prompt
	summary, err := c.backend.Summarize(ctx, prompt)
	if err != nil {
		return "", err
	}

	// Post-process: substitute user IDs with real names
	return c.substituteUserNames(summary, messages)
}

// substituteUserNames replaces user_ID placeholders with real names in the summary
func (c *Client) substituteUserNames(summary string, messages []database.MessageForSummary) (string, error) {
	// Build map of unique user IDs from messages
	userIDs := make(map[int64]struct{})
	for _, msg := range messages {
		userIDs[msg.UserID] = struct{}{}
	}

	// Substitute each user ID with real name from database
	for userID := range userIDs {
		if c.db != nil {
			if userName, err := c.db.GetUserNameByID(userID); err == nil {
				summary = strings.ReplaceAll(summary, fmt.Sprintf("user_%d", userID), userName)
			}
		}
	}

	return summary, nil
}

// GetBackend returns the underlying AI backend for type-specific operations
func (c *Client) GetBackend() AIClient {
	return c.backend
}
