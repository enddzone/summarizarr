package ai

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"summarizarr/internal/ollama"
	openaiclient "summarizarr/internal/openai"
)

// SummarizationPrompt is the template used for all LLM backends
const SummarizationPrompt = `Please provide a concise summary of this Signal group conversation using the following exact markdown format:

## Key topics discussed
- [List each main topic as a bullet point]

## Important decisions or conclusions
- [List each decision or conclusion as a bullet point]

## Action items or next steps
- [List each action item as a bullet point]

## Notable reactions or responses
- [List notable reactions as bullet points]

IMPORTANT: Use exactly the header format shown above (## Header name). Each section should be a proper markdown header followed by bullet points.

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

// SanitizeSummaryFormat ensures consistent markdown formatting for summaries
func SanitizeSummaryFormat(summary string) string {
	// Expected section headers in order
	expectedHeaders := []string{
		"Key topics discussed",
		"Important decisions or conclusions", 
		"Action items or next steps",
		"Notable reactions or responses",
	}
	
	// Normalize various header formats to consistent ## format
	for _, header := range expectedHeaders {
		// Match variations like "**Key topics discussed**", "Key topics discussed:", etc.
		patterns := []string{
			fmt.Sprintf(`(?i)\*\*%s\*\*:?`, regexp.QuoteMeta(header)),
			fmt.Sprintf(`(?i)%s:`, regexp.QuoteMeta(header)),
			fmt.Sprintf(`(?i)#{1,4}\s*%s`, regexp.QuoteMeta(header)),
		}
		
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			summary = re.ReplaceAllString(summary, fmt.Sprintf("## %s", header))
		}
	}
	
	// Fix nested lists - convert "- Topic:\n  - Detail" to proper format
	nestedListRe := regexp.MustCompile(`(?m)^(\s*)- ([^:]+):\s*\n(\s+)- (.+)$`)
	summary = nestedListRe.ReplaceAllString(summary, "- $2: $4")
	
	// Ensure proper spacing between sections
	headerRe := regexp.MustCompile(`(?m)^## (.+)$`)
	summary = headerRe.ReplaceAllString(summary, "\n## $1\n")
	
	// Clean up extra newlines
	summary = regexp.MustCompile(`\n{3,}`).ReplaceAllString(summary, "\n\n")
	summary = strings.TrimSpace(summary)
	
	return summary
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

	// Sanitize format for consistency
	summary = SanitizeSummaryFormat(summary)

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
