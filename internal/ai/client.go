package ai

import (
	"context"
	"fmt"
	"log/slog"
	"summarizarr/internal/database"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Client is a client for the OpenAI API.
type Client struct {
	*openai.Client
	model string
}

// NewClient creates a new OpenAI client.
func NewClient(apiKey, model string) *Client {
	return &Client{
		Client: openai.NewClient(apiKey),
		model:  model,
	}
}

// Summarize summarizes a list of messages with retry logic.
func (c *Client) Summarize(ctx context.Context, messages []database.MessageForSummary) (string, error) {
	var content string
	for _, msg := range messages {
		switch msg.MessageType {
		case "reaction":
			if msg.ReactionEmoji != "" {
				content += fmt.Sprintf("%s reacted with %s\n", msg.UserName, msg.ReactionEmoji)
			}
		case "quote":
			if msg.QuoteText != "" && msg.Text != "" {
				content += fmt.Sprintf("%s (replying to: \"%s\"): %s\n", msg.UserName, msg.QuoteText, msg.Text)
			} else if msg.Text != "" {
				content += fmt.Sprintf("%s: %s\n", msg.UserName, msg.Text)
			}
		default: // regular message
			if msg.Text != "" {
				content += fmt.Sprintf("%s: %s\n", msg.UserName, msg.Text)
			}
		}
	}

	// Retry logic for API calls
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := c.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model:       c.model,
				Temperature: 0.3, // Lower temperature for more consistent summaries
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: fmt.Sprintf("Please provide a concise summary of this Signal group conversation. Focus on:\n- Key topics discussed\n- Important decisions or conclusions\n- Action items or next steps\n- Notable reactions or responses\n\nConversation format: Regular messages, quoted replies (shown as 'replying to: \"original text\"'), and emoji reactions.\n\nConversation:\n%s", content),
					},
				},
			},
		)

		if err != nil {
			slog.Warn("OpenAI API call failed", "attempt", attempt+1, "error", err)
			if attempt == maxRetries-1 {
				return "", fmt.Errorf("failed to create chat completion after %d attempts: %w", maxRetries, err)
			}
			// Exponential backoff
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("no choices returned from AI service")
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("unexpected error in retry loop")
}
