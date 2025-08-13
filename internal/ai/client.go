package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"summarizarr/internal/llm"
	"summarizarr/internal/ollama"
	"time"
)

// Pre-compiled regex patterns to prevent ReDoS attacks
var (
	// Limit header length to 100 characters to prevent ReDoS
	boldHeaderRe  = regexp.MustCompile(`(?i)\*\*([^*\n]{1,100})\*\*:?`)
	colonHeaderRe = regexp.MustCompile(`(?i)^([^:\n]{1,100}):$`)
	hashHeaderRe  = regexp.MustCompile(`(?i)^#{1,4}\s*([^\n]{1,100})$`)
	// Limit nested list content to prevent ReDoS
	nestedListRe    = regexp.MustCompile(`(?m)^(\s{0,8})- ([^:\n]{1,100}):\s*\n(\s{1,12})- (.{1,500})$`)
	headerSpacingRe = regexp.MustCompile(`(?m)^## (.+)$`)
	extraNewlinesRe = regexp.MustCompile(`\n{3,}`)

	// Maximum input size to prevent DoS attacks
	maxSummarySize = 50 * 1024 // 50KB
	// Timeout for regex operations
	regexTimeout = 5 * time.Second

	// Standard timeout for all AI provider HTTP clients
	standardClientTimeout = 120 * time.Second
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

// SanitizeSummaryFormat ensures consistent markdown formatting for summaries with security safeguards
func SanitizeSummaryFormat(summary string) string {
	// Input validation to prevent DoS attacks
	if len(summary) > maxSummarySize {
		slog.Warn("Summary exceeds maximum size, truncating", "size", len(summary), "max", maxSummarySize)
		return "Summary too long to process safely"
	}

	if strings.TrimSpace(summary) == "" {
		return ""
	}

	// Add timeout context for regex operations
	ctx, cancel := context.WithTimeout(context.Background(), regexTimeout)
	defer cancel()

	// Check if context is cancelled during processing
	select {
	case <-ctx.Done():
		slog.Error("Regex processing timeout")
		return "Summary processing timeout"
	default:
	}

	// Expected section headers in order
	expectedHeaders := []string{
		"Key topics discussed",
		"Important decisions or conclusions",
		"Action items or next steps",
		"Notable reactions or responses",
	}

	// Normalize various header formats to consistent ## format using pre-compiled regex
	for _, header := range expectedHeaders {
		quotedHeader := regexp.QuoteMeta(header)

		// Use pre-compiled patterns with bounded replacements
		boldPattern := strings.Replace(boldHeaderRe.String(), `([^*\n]{1,100})`, fmt.Sprintf(`(%s)`, quotedHeader), 1)
		if boldRe, err := regexp.Compile(boldPattern); err == nil {
			summary = boldRe.ReplaceAllString(summary, fmt.Sprintf("## %s", header))
		}

		colonPattern := strings.Replace(colonHeaderRe.String(), `([^:\n]{1,100})`, fmt.Sprintf(`(%s)`, quotedHeader), 1)
		if colonRe, err := regexp.Compile(colonPattern); err == nil {
			summary = colonRe.ReplaceAllString(summary, fmt.Sprintf("## %s", header))
		}

		hashPattern := strings.Replace(hashHeaderRe.String(), `([^\n]{1,100})`, fmt.Sprintf(`(%s)`, quotedHeader), 1)
		if hashRe, err := regexp.Compile(hashPattern); err == nil {
			summary = hashRe.ReplaceAllString(summary, fmt.Sprintf("## %s", header))
		}

		// Check timeout again
		select {
		case <-ctx.Done():
			slog.Error("Regex processing timeout during header normalization")
			return "Summary processing timeout"
		default:
		}
	}

	// Fix nested lists using pre-compiled regex
	summary = nestedListRe.ReplaceAllString(summary, "- $2: $4")

	// Ensure proper spacing between sections
	summary = headerSpacingRe.ReplaceAllString(summary, "\n## $1\n")

	// Clean up extra newlines using pre-compiled regex
	summary = extraNewlinesRe.ReplaceAllString(summary, "\n\n")
	summary = strings.TrimSpace(summary)

	return summary
}

// Client wraps an AI backend client.
type Client struct {
	backend AIClient
	db      DB
}

// validateProviderConfig validates provider-specific configuration requirements
func validateProviderConfig(cfg *config.Config) error {
	provider := cfg.AIProvider

	switch provider {
	case "local":
		if cfg.OllamaHost == "" {
			return errors.New("OLLAMA_HOST is required for local provider")
		}
		if cfg.LocalModel == "" {
			return errors.New("LOCAL_MODEL is required for local provider")
		}
	case "openai":
		if cfg.OpenAIAPIKey == "" {
			return errors.New("OPENAI_API_KEY is required for openai provider")
		}
		if cfg.OpenAIModel == "" {
			return errors.New("OPENAI_MODEL is required for openai provider")
		}
	case "groq":
		if cfg.GroqAPIKey == "" {
			return errors.New("GROQ_API_KEY is required for groq provider")
		}
		if cfg.GroqModel == "" {
			return errors.New("GROQ_MODEL is required for groq provider")
		}
	case "gemini":
		if cfg.GeminiAPIKey == "" {
			return errors.New("GEMINI_API_KEY is required for gemini provider")
		}
		if cfg.GeminiModel == "" {
			return errors.New("GEMINI_MODEL is required for gemini provider")
		}
	case "claude":
		if cfg.ClaudeAPIKey == "" {
			return errors.New("CLAUDE_API_KEY is required for claude provider")
		}
		if cfg.ClaudeModel == "" {
			return errors.New("CLAUDE_MODEL is required for claude provider")
		}
	case "":
		return errors.New("AI_PROVIDER must be specified")
	default:
		return fmt.Errorf("unsupported AI provider: %s (supported: 'local', 'openai', 'groq', 'gemini', 'claude')", provider)
	}

	return nil
}

// NewClient creates a new AI client based on the configuration.
func NewClient(cfg *config.Config, db DB) (*Client, error) {
	// Validate configuration before creating client
	if err := validateProviderConfig(cfg); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	var backend AIClient
	provider := cfg.AIProvider

	switch provider {
	case "local":
		backend = ollama.NewClient(cfg.OllamaHost, cfg.LocalModel)
	case "openai":
		backend = llm.NewClient(llm.Config{
			APIKey:  cfg.OpenAIAPIKey,
			Model:   cfg.OpenAIModel,
			BaseURL: cfg.OpenAIBaseURL,
		})
	case "groq":
		backend = llm.NewClient(llm.Config{
			APIKey:  cfg.GroqAPIKey,
			Model:   cfg.GroqModel,
			BaseURL: cfg.GroqBaseURL,
		})
	case "gemini":
		backend = llm.NewClient(llm.Config{
			APIKey:  cfg.GeminiAPIKey,
			Model:   cfg.GeminiModel,
			BaseURL: cfg.GeminiBaseURL,
		})
	case "claude":
		backend = llm.NewClient(llm.Config{
			APIKey:  cfg.ClaudeAPIKey,
			Model:   cfg.ClaudeModel,
			BaseURL: cfg.ClaudeBaseURL,
		})
	default:
		// This should never be reached due to validation above, but keeping for safety
		return nil, fmt.Errorf("unsupported AI provider: %s (supported: 'local', 'openai', 'groq', 'gemini', 'claude')", provider)
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
		userPlaceholder := fmt.Sprintf("user_%d", userID)

		if c.db != nil {
			userName, err := c.db.GetUserNameByID(userID)
			if err != nil {
				// Log the error but continue with fallback behavior
				slog.Warn("Failed to get user name from database",
					"user_id", userID,
					"error", err.Error())

				// Fallback: use a generic "User <ID>" format instead of user_ID
				userName = fmt.Sprintf("User %d", userID)
			}

			// Replace user placeholder with actual name or fallback
			summary = strings.ReplaceAll(summary, userPlaceholder, userName)

			slog.Debug("Substituted user name",
				"user_id", userID,
				"placeholder", userPlaceholder,
				"name", userName)
		} else {
			// Database not available - log warning and use fallback
			slog.Warn("Database not available for user name substitution", "user_id", userID)
			fallbackName := fmt.Sprintf("User %d", userID)
			summary = strings.ReplaceAll(summary, userPlaceholder, fallbackName)
		}
	}

	return summary, nil
}

// GetBackend returns the underlying AI backend for type-specific operations
func (c *Client) GetBackend() AIClient {
	return c.backend
}
