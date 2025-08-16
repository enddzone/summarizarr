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
	// Handle host URL properly - don't add http:// if http:// or https:// is already present (supports both http and https)
	baseURL := host
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		baseURL = fmt.Sprintf("http://%s", host)
	}

	return &Client{
		baseURL: baseURL,
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

// ValidateExternalOllama performs comprehensive readiness checks for the external Ollama server
func (c *Client) ValidateExternalOllama(ctx context.Context) error {
	slog.InfoContext(ctx, "Validating external Ollama server", "baseURL", c.baseURL, "model", c.model)

	// Step 1: Check server connectivity
	if err := c.HealthCheck(ctx); err != nil {
		return fmt.Errorf("ollama server not accessible at %s: %w\n\nTroubleshooting:\n- Ensure Ollama is running: 'ollama serve'\n- Check if port 11434 is accessible\n- Verify no firewall blocking connection", c.baseURL, err)
	}

	// Step 2: Verify model exists
	if err := c.CheckModelExists(ctx, c.model); err != nil {
		return fmt.Errorf("model validation failed: %w\n\nTroubleshooting:\n- Pull the model: 'ollama pull %s'\n- List available models: 'ollama list'\n- Check model name spelling", err, c.model)
	}

	// Step 3: Test model inference
	if err := c.TestModelInference(ctx); err != nil {
		return fmt.Errorf("model inference test failed: %w\n\nTroubleshooting:\n- Model may be corrupted, try: 'ollama pull %s'\n- Check available system resources (RAM/GPU)\n- Verify model is compatible with your system", err, c.model)
	}

	slog.InfoContext(ctx, "External Ollama validation successful", "model", c.model)
	return nil
}

// CheckModelExists verifies that the specified model is available in the Ollama server
func (c *Client) CheckModelExists(ctx context.Context, modelName string) error {
	models, err := c.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	// Check if our model is available
	for _, model := range models.Models {
		if strings.Contains(model.Name, modelName) {
			slog.InfoContext(ctx, "Model found", "model", modelName, "size_mb", model.Size/1024/1024)
			return nil
		}
	}

	// Provide helpful error with available models
	var availableModels []string
	for _, model := range models.Models {
		availableModels = append(availableModels, model.Name)
	}

	if len(availableModels) == 0 {
		return fmt.Errorf("model '%s' not found and no models are available", modelName)
	}

	return fmt.Errorf("model '%s' not found. Available models: %s", modelName, strings.Join(availableModels, ", "))
}

// TestModelInference validates that the model can successfully process requests
func (c *Client) TestModelInference(ctx context.Context) error {
	slog.InfoContext(ctx, "Testing model inference", "model", c.model)

	// Create a simple test request with a short timeout
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	chatReq := ChatCompletionRequest{
		Model: c.model,
		Messages: []ChatCompletionMessage{
			{
				Role:    "user",
				Content: "Say 'test'",
			},
		},
		Stream: false,
	}

	response, err := c.performChatCompletion(testCtx, chatReq)
	if err != nil {
		return fmt.Errorf("model inference test failed: %w", err)
	}

	if len(strings.TrimSpace(response)) == 0 {
		return fmt.Errorf("model returned empty response")
	}

	slog.InfoContext(ctx, "Model inference test successful", "model", c.model, "response_length", len(response))
	return nil
}
