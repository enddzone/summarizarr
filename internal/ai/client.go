package ai

import (
	"context"
	"fmt"
	"summarizarr/internal/database"

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

// Summarize summarizes a list of messages.
func (c *Client) Summarize(ctx context.Context, messages []database.MessageForSummary) (string, error) {
	var content string
	for _, msg := range messages {
		content += fmt.Sprintf("%s: %s\n", msg.UserName, msg.Text)
	}

	resp, err := c.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Please summarize the following conversation. Identify the key topics, any decisions that were made, and any action items. For each point, please indicate which user made the point.\n\n%s", content),
				},
			},
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from AI service")
	}

	return resp.Choices[0].Message.Content, nil
}