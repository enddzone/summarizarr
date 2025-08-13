package openai

import (
	context "context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	apiKey string
	model  string
	client *openai.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		client: openai.NewClient(apiKey),
	}
}

func (c *Client) Summarize(ctx context.Context, prompt string) (string, error) {
	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{{
			Role:    "user",
			Content: prompt,
		}},
	})
	if err != nil {
		return "", err
	}

	// Validate response has choices before accessing
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from AI provider")
	}

	// Validate choice has content
	choice := resp.Choices[0]
	if choice.Message.Content == "" {
		return "", fmt.Errorf("empty response content from AI provider")
	}

	return choice.Message.Content, nil
}
