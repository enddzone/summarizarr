package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"testing"
	"time"
)

// MockOpenAIResponse represents a mock OpenAI API response
type MockOpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// MockOpenAIRequest represents a mock OpenAI API request
type MockOpenAIRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

// createMockOpenAIServer creates a mock HTTP server that simulates an OpenAI-compatible API
func createMockOpenAIServer(t *testing.T, expectedModel string, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Validate request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Validate request headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type")
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Bearer token in Authorization header, got %s", authHeader)
		}

		// Parse and validate request body
		var req MockOpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate model
		if req.Model != expectedModel {
			t.Errorf("Expected model %s, got %s", expectedModel, req.Model)
		}

		// Validate messages structure
		if len(req.Messages) != 1 {
			t.Errorf("Expected 1 message, got %d", len(req.Messages))
		}

		if req.Messages[0].Role != "user" {
			t.Errorf("Expected user role, got %s", req.Messages[0].Role)
		}

		// Create mock response
		mockResp := MockOpenAIResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   expectedModel,
			Choices: []struct {
				Index   int `json:"index"`
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Index: 0,
					Message: struct {
						Role    string `json:"role"`
						Content string `json:"content"`
					}{
						Role:    "assistant",
						Content: response,
					},
					FinishReason: "stop",
				},
			},
			Usage: struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			}{
				PromptTokens:     100,
				CompletionTokens: 50,
				TotalTokens:      150,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResp)
	}))
}

// TestOpenAIProviderIntegration tests OpenAI provider with mock server
func TestOpenAIProviderIntegration(t *testing.T) {
	mockResponse := `## Key topics discussed
- Project updates and progress
- Team coordination

## Important decisions or conclusions
- Approved next phase of development

## Action items or next steps
- Schedule follow-up meeting

## Notable reactions or responses
- Team expressed enthusiasm`

	server := createMockOpenAIServer(t, "gpt-4", mockResponse)
	defer server.Close()

	cfg := &config.Config{
		AIProvider:    "openai",
		OpenAIAPIKey:  "sk-test-key",
		OpenAIModel:   "gpt-4",
		OpenAIBaseURL: server.URL,
	}

	mockDB := &MockDB{
		shouldError: false,
		users: map[int64]string{
			123: "Alice",
			456: "Bob",
		},
	}

	client, err := NewClient(cfg, mockDB)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	messages := []database.MessageForSummary{
		{UserID: 123, Text: "Let's discuss the project"},
		{UserID: 456, Text: "Sounds good, what's the status?"},
	}

	ctx := context.Background()
	summary, err := client.Summarize(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Verify the response contains expected sections
	expectedSections := []string{
		"## Key topics discussed",
		"## Important decisions or conclusions",
		"## Action items or next steps",
		"## Notable reactions or responses",
	}

	for _, section := range expectedSections {
		if !strings.Contains(summary, section) {
			t.Errorf("Expected summary to contain section %q", section)
		}
	}
}

// TestGroqProviderIntegration tests Groq provider with mock server
func TestGroqProviderIntegration(t *testing.T) {
	mockResponse := `## Key topics discussed
- Development roadmap discussion
- Technical challenges review

## Important decisions or conclusions
- Decided on new architecture approach

## Action items or next steps
- Begin prototype development

## Notable reactions or responses
- Positive feedback on proposed solution`

	server := createMockOpenAIServer(t, "llama3-8b-8192", mockResponse)
	defer server.Close()

	cfg := &config.Config{
		AIProvider:  "groq",
		GroqAPIKey:  "gsk-test-key",
		GroqModel:   "llama3-8b-8192",
		GroqBaseURL: server.URL,
	}

	mockDB := &MockDB{shouldError: false}

	client, err := NewClient(cfg, mockDB)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	messages := []database.MessageForSummary{
		{UserID: 789, Text: "What's our development approach?"},
		{UserID: 101, Text: "I think we should consider a new architecture"},
	}

	ctx := context.Background()
	summary, err := client.Summarize(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Verify Groq-specific response processing
	if !strings.Contains(summary, "architecture approach") {
		t.Errorf("Expected summary to contain Groq response content")
	}
}

// TestGeminiProviderIntegration tests Gemini provider via proxy with mock server
func TestGeminiProviderIntegration(t *testing.T) {
	mockResponse := `## Key topics discussed
- AI model evaluation and selection
- Performance benchmarking results

## Important decisions or conclusions
- Gemini model shows promising results

## Action items or next steps
- Conduct more extensive testing

## Notable reactions or responses
- Team impressed with model capabilities`

	server := createMockOpenAIServer(t, "gemini-2.0-flash", mockResponse)
	defer server.Close()

	cfg := &config.Config{
		AIProvider:    "gemini",
		GeminiAPIKey:  "test-key",
		GeminiModel:   "gemini-2.0-flash",
		GeminiBaseURL: server.URL,
	}

	mockDB := &MockDB{shouldError: false}

	client, err := NewClient(cfg, mockDB)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	messages := []database.MessageForSummary{
		{UserID: 111, Text: "How's the Gemini model performing?"},
		{UserID: 222, Text: "The benchmarks look really good"},
	}

	ctx := context.Background()
	summary, err := client.Summarize(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Verify Gemini-specific response processing
	if !strings.Contains(summary, "Gemini model") {
		t.Errorf("Expected summary to contain Gemini response content")
	}
}

// TestClaudeProviderIntegration tests Claude provider via proxy with mock server
func TestClaudeProviderIntegration(t *testing.T) {
	mockResponse := `## Key topics discussed
- Claude AI integration strategy
- API compatibility considerations

## Important decisions or conclusions
- Claude provides excellent reasoning capabilities

## Action items or next steps
- Implement Claude integration

## Notable reactions or responses
- Team excited about Claude's potential`

	server := createMockOpenAIServer(t, "claude-3-sonnet", mockResponse)
	defer server.Close()

	cfg := &config.Config{
		AIProvider:    "claude",
		ClaudeAPIKey:  "sk-ant-test-key",
		ClaudeModel:   "claude-3-sonnet",
		ClaudeBaseURL: server.URL,
	}

	mockDB := &MockDB{shouldError: false}

	client, err := NewClient(cfg, mockDB)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	messages := []database.MessageForSummary{
		{UserID: 333, Text: "Should we integrate Claude AI?"},
		{UserID: 444, Text: "Claude has great reasoning abilities"},
	}

	ctx := context.Background()
	summary, err := client.Summarize(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Verify Claude-specific response processing
	if !strings.Contains(summary, "reasoning capabilities") {
		t.Errorf("Expected summary to contain Claude response content")
	}
}

// TestProviderErrorHandling tests error scenarios for each provider
func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		serverError bool
		expectedErr string
	}{
		{
			name:        "OpenAI server error",
			provider:    "openai",
			serverError: true,
			expectedErr: "500",
		},
		{
			name:        "Groq server error",
			provider:    "groq",
			serverError: true,
			expectedErr: "500",
		},
		{
			name:        "Gemini server error",
			provider:    "gemini",
			serverError: true,
			expectedErr: "500",
		},
		{
			name:        "Claude server error",
			provider:    "claude",
			serverError: true,
			expectedErr: "500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverError {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
					return
				}
			}))
			defer server.Close()

			var cfg *config.Config
			switch tt.provider {
			case "openai":
				cfg = &config.Config{
					AIProvider:    "openai",
					OpenAIAPIKey:  "sk-test",
					OpenAIModel:   "gpt-4",
					OpenAIBaseURL: server.URL,
				}
			case "groq":
				cfg = &config.Config{
					AIProvider:  "groq",
					GroqAPIKey:  "gsk-test",
					GroqModel:   "llama3-8b-8192",
					GroqBaseURL: server.URL,
				}
			case "gemini":
				cfg = &config.Config{
					AIProvider:    "gemini",
					GeminiAPIKey:  "test-key",
					GeminiModel:   "gemini-2.0-flash",
					GeminiBaseURL: server.URL,
				}
			case "claude":
				cfg = &config.Config{
					AIProvider:    "claude",
					ClaudeAPIKey:  "sk-ant-test",
					ClaudeModel:   "claude-3-sonnet",
					ClaudeBaseURL: server.URL,
				}
			}

			mockDB := &MockDB{shouldError: false}
			client, err := NewClient(cfg, mockDB)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			messages := []database.MessageForSummary{
				{UserID: 1, Text: "Test message"},
			}

			ctx := context.Background()
			_, err = client.Summarize(ctx, messages)

			if tt.serverError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.expectedErr) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedErr, err.Error())
				}
			}
		})
	}
}

// TestRequestTimeouts tests that all providers handle timeouts appropriately
func TestRequestTimeouts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	providers := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "openai",
			cfg: &config.Config{
				AIProvider:    "openai",
				OpenAIAPIKey:  "sk-test",
				OpenAIModel:   "gpt-4",
				OpenAIBaseURL: slowServer.URL,
			},
		},
		{
			name: "groq",
			cfg: &config.Config{
				AIProvider:  "groq",
				GroqAPIKey:  "gsk-test",
				GroqModel:   "llama3-8b-8192",
				GroqBaseURL: slowServer.URL,
			},
		},
		{
			name: "gemini",
			cfg: &config.Config{
				AIProvider:    "gemini",
				GeminiAPIKey:  "test-key",
				GeminiModel:   "gemini-2.0-flash",
				GeminiBaseURL: slowServer.URL,
			},
		},
		{
			name: "claude",
			cfg: &config.Config{
				AIProvider:    "claude",
				ClaudeAPIKey:  "sk-ant-test",
				ClaudeModel:   "claude-3-sonnet",
				ClaudeBaseURL: slowServer.URL,
			},
		},
	}

	for _, provider := range providers {
		t.Run(provider.name+"_timeout", func(t *testing.T) {
			mockDB := &MockDB{shouldError: false}
			client, err := NewClient(provider.cfg, mockDB)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			messages := []database.MessageForSummary{
				{UserID: 1, Text: "Test message"},
			}

			// Create context with short timeout
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			_, err = client.Summarize(ctx, messages)

			// Should get a timeout or context cancelled error
			if err == nil {
				t.Errorf("Expected timeout error but got none")
			} else if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "timeout") {
				t.Errorf("Expected timeout/context error, got: %v", err)
			}
		})
	}
}

// TestConcurrentRequests tests that all providers handle concurrent requests properly
func TestConcurrentRequests(t *testing.T) {
	mockResponse := `## Key topics discussed
- Concurrent processing test

## Important decisions or conclusions
- System handles concurrent requests

## Action items or next steps
- Continue testing

## Notable reactions or responses
- Positive test results`

	server := createMockOpenAIServer(t, "gpt-4", mockResponse)
	defer server.Close()

	cfg := &config.Config{
		AIProvider:    "openai",
		OpenAIAPIKey:  "sk-test",
		OpenAIModel:   "gpt-4",
		OpenAIBaseURL: server.URL,
	}

	mockDB := &MockDB{shouldError: false}
	client, err := NewClient(cfg, mockDB)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	messages := []database.MessageForSummary{
		{UserID: 1, Text: "Concurrent test message"},
	}

	// Run multiple concurrent requests
	numRequests := 5
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			ctx := context.Background()
			_, err := client.Summarize(ctx, messages)
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}
}
