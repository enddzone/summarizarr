package openai

import (
	context "context"

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

	return resp.Choices[0].Message.Content, nil
}
