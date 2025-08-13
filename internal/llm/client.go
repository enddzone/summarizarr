package llm

import (
	"context"
	"fmt"
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// Standard timeout for all AI provider HTTP clients - matches Ollama client
	StandardClientTimeout = 120 * time.Second
)

// Client is a generic OpenAI-compatible client that can work with multiple providers
type Client struct {
	apiKey  string
	model   string
	baseURL string
	client  *openai.Client
}

// Config holds the configuration for an OpenAI-compatible provider
type Config struct {
	APIKey  string
	Model   string
	BaseURL string
	Timeout time.Duration // Optional timeout, defaults to standard client timeout
}

// NewClient creates a new OpenAI-compatible client with configurable base URL and timeout
func NewClient(config Config) *Client {
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	// Use configured timeout or default to standard timeout
	timeout := config.Timeout
	if timeout == 0 {
		timeout = StandardClientTimeout
	}

	// Configure HTTP client with standardized timeout
	clientConfig.HTTPClient = &http.Client{
		Timeout: timeout,
	}

	return &Client{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: config.BaseURL,
		client:  openai.NewClientWithConfig(clientConfig),
	}
}

// Summarize generates a summary using the configured OpenAI-compatible provider
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
