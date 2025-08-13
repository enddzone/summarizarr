package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	// Standard timeout for AI provider HTTP clients
	StandardClientTimeout = 120 * time.Second
)

// Client provides an OpenAI-compatible interface for local Ollama models
type Client struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewClient creates a new Ollama client
func NewClient(host, model string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s", host),
		model:   model,
		client: &http.Client{
			Timeout: StandardClientTimeout, // Standardized timeout for model downloads and inference
		},
	}
}

// ChatCompletionMessage represents a message in the conversation
type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	Temperature float32                 `json:"temperature,omitempty"`
	Stream      bool                    `json:"stream"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// PullRequest represents a model pull request
type PullRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
}

// PullResponse represents a model pull response
type PullResponse struct {
	Status    string `json:"status"`
	Digest    string `json:"digest,omitempty"`
	Total     int64  `json:"total,omitempty"`
	Completed int64  `json:"completed,omitempty"`
}

// ListModelsResponse represents the response from listing models
type ListModelsResponse struct {
	Models []Model `json:"models"`
}

// Model represents an Ollama model
type Model struct {
	Name      string       `json:"name"`
	Model     string       `json:"model"`
	Size      int64        `json:"size"`
	Digest    string       `json:"digest"`
	Details   ModelDetails `json:"details"`
	ExpiresAt time.Time    `json:"expires_at,omitempty"`
}

// ModelDetails contains model metadata
type ModelDetails struct {
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

// EnsureModel downloads the model if it doesn't exist locally
func (c *Client) EnsureModel(ctx context.Context, autoDownload bool) error {
	// First check if model exists
	models, err := c.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Check if our model is already available
	for _, model := range models.Models {
		if strings.Contains(model.Name, c.model) {
			slog.InfoContext(ctx, "Model already available", "model", c.model)
			return nil
		}
	}

	if !autoDownload {
		return fmt.Errorf("model %s not found and auto-download is disabled", c.model)
	}

	// Download the model
	slog.InfoContext(ctx, "Model not found, downloading...", "model", c.model)
	return c.PullModel(ctx, c.model)
}

// PullModel downloads a model from the Ollama registry with retry logic
func (c *Client) PullModel(ctx context.Context, model string) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt) * 30 * time.Second
			slog.InfoContext(ctx, "Retrying model download", "attempt", attempt+1, "wait_seconds", int(waitTime.Seconds()))

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		lastErr = c.pullModelAttempt(ctx, model)
		if lastErr == nil {
			slog.InfoContext(ctx, "Model downloaded successfully", "model", model)
			return nil
		}

		slog.WarnContext(ctx, "Model download attempt failed", "attempt", attempt+1, "error", lastErr)
	}

	return fmt.Errorf("failed to download model after %d attempts: %w", maxRetries, lastErr)
}

// pullModelAttempt performs a single attempt to download a model
func (c *Client) pullModelAttempt(ctx context.Context, model string) error {
	pullReq := PullRequest{
		Model:  model,
		Stream: true,
	}

	body, err := json.Marshal(pullReq)
	if err != nil {
		return fmt.Errorf("failed to marshal pull request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send pull request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pull request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Process streaming response
	decoder := json.NewDecoder(resp.Body)
	var lastStatus string
	var startTime = time.Now()

	for {
		var pullResp PullResponse
		if err := decoder.Decode(&pullResp); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode pull response: %w", err)
		}

		// Log progress for different phases
		if pullResp.Status != lastStatus {
			slog.InfoContext(ctx, "Model download progress", "status", pullResp.Status, "model", model)
			lastStatus = pullResp.Status
		}

		// Log download progress with estimated time remaining
		if pullResp.Total > 0 && pullResp.Completed > 0 {
			progress := float64(pullResp.Completed) / float64(pullResp.Total) * 100
			elapsed := time.Since(startTime)

			var eta string
			if progress > 0 {
				totalEstimated := time.Duration(float64(elapsed) / (progress / 100))
				remaining := totalEstimated - elapsed
				eta = fmt.Sprintf("ETA: %v", remaining.Round(time.Second))
			}

			slog.InfoContext(ctx, "Download progress",
				"model", model,
				"percent", fmt.Sprintf("%.1f%%", progress),
				"downloaded_mb", pullResp.Completed/1024/1024,
				"total_mb", pullResp.Total/1024/1024,
				"eta", eta)
		}

		// Check for success
		if pullResp.Status == "success" {
			return nil
		}
	}

	return fmt.Errorf("model download completed but success status not received")
}

// ListModels lists all available models
func (c *Client) ListModels(ctx context.Context) (*ListModelsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send list request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list request failed with status: %d", resp.StatusCode)
	}

	var listResp ListModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	return &listResp, nil
}

// Summarize generates a summary using the local model with the provided prompt
func (c *Client) Summarize(ctx context.Context, prompt string) (string, error) {
	// Create chat completion request
	chatReq := ChatCompletionRequest{
		Model:       c.model,
		Temperature: 0.3, // Lower temperature for more consistent summaries
		Stream:      false,
		Messages: []ChatCompletionMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Retry logic for API calls
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			waitTime := time.Duration(attempt) * 5 * time.Second
			slog.InfoContext(ctx, "Retrying summarization", "attempt", attempt+1, "wait_seconds", int(waitTime.Seconds()))

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(waitTime):
			}
		}

		summary, err := c.performChatCompletion(ctx, chatReq)
		if err == nil {
			return summary, nil
		}

		slog.WarnContext(ctx, "Summarization attempt failed", "attempt", attempt+1, "error", err)

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
	}

	return "", fmt.Errorf("failed to generate summary after %d attempts", maxRetries)
}

// performChatCompletion executes a single chat completion request
func (c *Client) performChatCompletion(ctx context.Context, chatReq ChatCompletionRequest) (string, error) {
	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("chat request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode chat response: %w", err)
	}

	if !chatResp.Done {
		return "", fmt.Errorf("chat completion not finished")
	}

	return strings.TrimSpace(chatResp.Message.Content), nil
}

// HealthCheck verifies that the Ollama server is responsive
func (c *Client) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// WarmupModel loads the model into memory by sending a simple chat completion request
func (c *Client) WarmupModel(ctx context.Context) error {
	slog.Info("Warming up model...", "model", c.model)

	// Create a simple chat completion request to warm up the model
	chatReq := ChatCompletionRequest{
		Model: c.model,
		Messages: []ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Hello",
			},
		},
		Stream: false,
	}

	_, err := c.performChatCompletion(ctx, chatReq)
	if err != nil {
		return fmt.Errorf("model warmup failed: %w", err)
	}

	slog.Info("Model warmed up successfully", "model", c.model)
	return nil
}
