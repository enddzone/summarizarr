package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"testing"
)

// ProviderTestCase represents a test case for provider-specific validation
type ProviderTestCase struct {
	name            string
	provider        string
	expectedModel   string
	expectedAPIKey  string
	expectedBaseURL string
	cfg             *config.Config
	validateRequest func(t *testing.T, r *http.Request)
}

// TestProviderRequestFormat validates that each provider sends correctly formatted requests
func TestProviderRequestFormat(t *testing.T) {
	testCases := []ProviderTestCase{
		{
			name:            "OpenAI request format validation",
			provider:        "openai",
			expectedModel:   "gpt-4-turbo",
			expectedAPIKey:  "sk-test-openai",
			expectedBaseURL: "",
			cfg: &config.Config{
				AIProvider:   "openai",
				OpenAIAPIKey: "sk-test-openai",
				OpenAIModel:  "gpt-4-turbo",
			},
			validateRequest: func(t *testing.T, r *http.Request) {
				// Validate OpenAI-specific request format
				validateOpenAIRequest(t, r, "gpt-4-turbo", "sk-test-openai")
			},
		},
		{
			name:            "Groq request format validation",
			provider:        "groq",
			expectedModel:   "mixtral-8x7b-32768",
			expectedAPIKey:  "gsk-test-groq",
			expectedBaseURL: "",
			cfg: &config.Config{
				AIProvider: "groq",
				GroqAPIKey: "gsk-test-groq",
				GroqModel:  "mixtral-8x7b-32768",
			},
			validateRequest: func(t *testing.T, r *http.Request) {
				// Groq uses OpenAI-compatible format
				validateOpenAIRequest(t, r, "mixtral-8x7b-32768", "gsk-test-groq")

				// Validate Groq-specific considerations
				if !strings.Contains(r.Header.Get("User-Agent"), "go-openai") {
					t.Logf("User-Agent: %s", r.Header.Get("User-Agent"))
				}
			},
		},
		{
			name:            "Gemini request format validation",
			provider:        "gemini",
			expectedModel:   "gemini-2.0-flash",
			expectedAPIKey:  "test-gemini-key",
			expectedBaseURL: "http://localhost:8000/hf/v1",
			cfg: &config.Config{
				AIProvider:    "gemini",
				GeminiAPIKey:  "test-gemini-key",
				GeminiModel:   "gemini-2.0-flash",
				GeminiBaseURL: "http://localhost:8000/hf/v1",
			},
			validateRequest: func(t *testing.T, r *http.Request) {
				// Gemini via proxy should use OpenAI-compatible format
				validateOpenAIRequest(t, r, "gemini-2.0-flash", "test-gemini-key")

				// Validate that the request goes to the proxy
				if r.Host == "api.openai.com" {
					t.Errorf("Expected request to go to proxy, but went to OpenAI directly")
				}
			},
		},
		{
			name:            "Claude request format validation",
			provider:        "claude",
			expectedModel:   "claude-3-opus",
			expectedAPIKey:  "sk-ant-test-claude",
			expectedBaseURL: "http://localhost:8000/openai/v1",
			cfg: &config.Config{
				AIProvider:    "claude",
				ClaudeAPIKey:  "sk-ant-test-claude",
				ClaudeModel:   "claude-3-opus",
				ClaudeBaseURL: "http://localhost:8000/openai/v1",
			},
			validateRequest: func(t *testing.T, r *http.Request) {
				// Claude via proxy should use OpenAI-compatible format
				validateOpenAIRequest(t, r, "claude-3-opus", "sk-ant-test-claude")

				// Validate that the request goes to the proxy
				if r.Host == "api.openai.com" {
					t.Errorf("Expected request to go to proxy, but went to OpenAI directly")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Run provider-specific request validation
				tc.validateRequest(t, r)

				// Return a mock response
				mockResp := MockOpenAIResponse{
					ID:      "chatcmpl-test",
					Object:  "chat.completion",
					Created: 1677652288,
					Model:   tc.expectedModel,
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
								Content: "## Key topics discussed\n- Test topic",
							},
							FinishReason: "stop",
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(mockResp); err != nil {
					// Fail the parent test if encoding fails
					t.Errorf("Failed to encode mock response: %v", err)
				}
			}))
			defer server.Close()

			// Update config with test server URL
			switch tc.provider {
			case "openai":
				tc.cfg.OpenAIBaseURL = server.URL
			case "groq":
				tc.cfg.GroqBaseURL = server.URL
			case "gemini":
				tc.cfg.GeminiBaseURL = server.URL
			case "claude":
				tc.cfg.ClaudeBaseURL = server.URL
			}

			mockDB := &MockDB{shouldError: false}
			client, err := NewClient(tc.cfg, mockDB)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			messages := []database.MessageForSummary{
				{UserID: 1, Text: "Test message for provider validation"},
			}

			ctx := context.Background()
			summary, err := client.Summarize(ctx, messages)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if summary == "" {
				t.Errorf("Expected non-empty summary")
			}
		})
	}
}

// validateOpenAIRequest validates common OpenAI-compatible request format
func validateOpenAIRequest(t *testing.T, r *http.Request, expectedModel, expectedAPIKey string) {
	// Validate HTTP method
	if r.Method != "POST" {
		t.Errorf("Expected POST method, got %s", r.Method)
	}

	// Validate path
	if r.URL.Path != "/chat/completions" {
		t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
	}

	// Validate headers
	if r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected application/json content type, got %s", r.Header.Get("Content-Type"))
	}

	authHeader := r.Header.Get("Authorization")
	expectedAuth := fmt.Sprintf("Bearer %s", expectedAPIKey)
	if authHeader != expectedAuth {
		t.Errorf("Expected Authorization header %q, got %q", expectedAuth, authHeader)
	}

	// Validate request body
	var req MockOpenAIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Errorf("Failed to decode request body: %v", err)
		return
	}

	if req.Model != expectedModel {
		t.Errorf("Expected model %q, got %q", expectedModel, req.Model)
	}

	if len(req.Messages) == 0 {
		t.Errorf("Expected at least one message in request")
	}

	if req.Messages[0].Role != "user" {
		t.Errorf("Expected first message role to be 'user', got %q", req.Messages[0].Role)
	}

	// Validate that the prompt contains our summarization template
	if !strings.Contains(req.Messages[0].Content, "Please provide a concise summary") {
		t.Errorf("Expected request to contain summarization prompt")
	}
}

// TestProviderResponseFormat validates that each provider's responses are handled correctly
func TestProviderResponseFormat(t *testing.T) {
	testCases := []struct {
		name            string
		provider        string
		mockResponse    MockOpenAIResponse
		expectedContent string
		validateContent func(t *testing.T, summary string)
	}{
		{
			name:     "OpenAI response format validation",
			provider: "openai",
			mockResponse: MockOpenAIResponse{
				ID:      "chatcmpl-openai-test",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "gpt-4",
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
							Content: "## Key topics discussed\n- OpenAI integration\n\n## Important decisions or conclusions\n- OpenAI format validated",
						},
						FinishReason: "stop",
					},
				},
			},
			expectedContent: "OpenAI integration",
			validateContent: func(t *testing.T, summary string) {
				if !strings.Contains(summary, "OpenAI integration") {
					t.Errorf("Expected summary to contain OpenAI-specific content")
				}
			},
		},
		{
			name:     "Groq response format validation",
			provider: "groq",
			mockResponse: MockOpenAIResponse{
				ID:      "chatcmpl-groq-test",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "mixtral-8x7b-32768",
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
							Content: "## Key topics discussed\n- Groq fast inference\n\n## Important decisions or conclusions\n- Groq provides excellent speed",
						},
						FinishReason: "stop",
					},
				},
			},
			expectedContent: "Groq fast inference",
			validateContent: func(t *testing.T, summary string) {
				if !strings.Contains(summary, "Groq") {
					t.Errorf("Expected summary to contain Groq-specific content")
				}
			},
		},
		{
			name:     "Gemini response format validation",
			provider: "gemini",
			mockResponse: MockOpenAIResponse{
				ID:      "chatcmpl-gemini-test",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "gemini-2.0-flash",
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
							Content: "## Key topics discussed\n- Gemini multimodal capabilities\n\n## Important decisions or conclusions\n- Gemini excels at reasoning",
						},
						FinishReason: "stop",
					},
				},
			},
			expectedContent: "multimodal capabilities",
			validateContent: func(t *testing.T, summary string) {
				if !strings.Contains(summary, "multimodal") {
					t.Errorf("Expected summary to contain Gemini-specific content")
				}
			},
		},
		{
			name:     "Claude response format validation",
			provider: "claude",
			mockResponse: MockOpenAIResponse{
				ID:      "chatcmpl-claude-test",
				Object:  "chat.completion",
				Created: 1677652288,
				Model:   "claude-3-opus",
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
							Content: "## Key topics discussed\n- Claude reasoning excellence\n\n## Important decisions or conclusions\n- Claude provides thoughtful analysis",
						},
						FinishReason: "stop",
					},
				},
			},
			expectedContent: "reasoning excellence",
			validateContent: func(t *testing.T, summary string) {
				if !strings.Contains(summary, "reasoning") {
					t.Errorf("Expected summary to contain Claude-specific content")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				if err := json.NewEncoder(w).Encode(tc.mockResponse); err != nil {
					t.Errorf("Failed to encode mock response: %v", err)
				}
			}))
			defer server.Close()

			var cfg *config.Config
			switch tc.provider {
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
					GroqModel:   "mixtral-8x7b-32768",
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
					ClaudeModel:   "claude-3-opus",
					ClaudeBaseURL: server.URL,
				}
			}

			mockDB := &MockDB{shouldError: false}
			client, err := NewClient(cfg, mockDB)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			messages := []database.MessageForSummary{
				{UserID: 1, Text: "Test message for response validation"},
			}

			ctx := context.Background()
			summary, err := client.Summarize(ctx, messages)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if summary == "" {
				t.Errorf("Expected non-empty summary")
			}

			// Run provider-specific content validation
			tc.validateContent(t, summary)
		})
	}
}

// TestProviderErrorResponses validates error handling for each provider
func TestProviderErrorResponses(t *testing.T) {
	errorTestCases := []struct {
		name        string
		provider    string
		statusCode  int
		errorBody   string
		expectedErr string
	}{
		{
			name:        "OpenAI authentication error",
			provider:    "openai",
			statusCode:  401,
			errorBody:   `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			expectedErr: "401",
		},
		{
			name:        "OpenAI rate limit error",
			provider:    "openai",
			statusCode:  429,
			errorBody:   `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectedErr: "429",
		},
		{
			name:        "Groq authentication error",
			provider:    "groq",
			statusCode:  401,
			errorBody:   `{"error": {"message": "Invalid API key", "type": "invalid_request_error"}}`,
			expectedErr: "401",
		},
		{
			name:        "Groq model not found error",
			provider:    "groq",
			statusCode:  404,
			errorBody:   `{"error": {"message": "Model not found", "type": "not_found_error"}}`,
			expectedErr: "404",
		},
		{
			name:        "Gemini proxy error",
			provider:    "gemini",
			statusCode:  502,
			errorBody:   `{"error": {"message": "Bad gateway", "type": "proxy_error"}}`,
			expectedErr: "502",
		},
		{
			name:        "Claude proxy error",
			provider:    "claude",
			statusCode:  503,
			errorBody:   `{"error": {"message": "Service unavailable", "type": "proxy_error"}}`,
			expectedErr: "503",
		},
	}

	for _, tc := range errorTestCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				if _, err := w.Write([]byte(tc.errorBody)); err != nil {
					t.Errorf("Failed to write error body: %v", err)
				}
			}))
			defer server.Close()

			var cfg *config.Config
			switch tc.provider {
			case "openai":
				cfg = &config.Config{
					AIProvider:    "openai",
					OpenAIAPIKey:  "sk-invalid",
					OpenAIModel:   "gpt-4",
					OpenAIBaseURL: server.URL,
				}
			case "groq":
				cfg = &config.Config{
					AIProvider:  "groq",
					GroqAPIKey:  "gsk-invalid",
					GroqModel:   "invalid-model",
					GroqBaseURL: server.URL,
				}
			case "gemini":
				cfg = &config.Config{
					AIProvider:    "gemini",
					GeminiAPIKey:  "invalid-key",
					GeminiModel:   "gemini-2.0-flash",
					GeminiBaseURL: server.URL,
				}
			case "claude":
				cfg = &config.Config{
					AIProvider:    "claude",
					ClaudeAPIKey:  "sk-ant-invalid",
					ClaudeModel:   "claude-3-opus",
					ClaudeBaseURL: server.URL,
				}
			}

			mockDB := &MockDB{shouldError: false}
			client, err := NewClient(cfg, mockDB)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			messages := []database.MessageForSummary{
				{UserID: 1, Text: "Test error message"},
			}

			ctx := context.Background()
			_, err = client.Summarize(ctx, messages)

			if err == nil {
				t.Errorf("Expected error but got none")
			} else if !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("Expected error containing %q, got %q", tc.expectedErr, err.Error())
			}
		})
	}
}

// TestProviderSpecificFeatures validates provider-specific features and limitations
func TestProviderSpecificFeatures(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "OpenAI supports all models",
			provider:    "openai",
			description: "OpenAI should support all standard models",
			testFunc: func(t *testing.T) {
				models := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"}
				for _, model := range models {
					cfg := &config.Config{
						AIProvider:   "openai",
						OpenAIAPIKey: "sk-test",
						OpenAIModel:  model,
					}
					mockDB := &MockDB{shouldError: false}
					client, err := NewClient(cfg, mockDB)
					if err != nil {
						t.Errorf("Failed to create OpenAI client with model %s: %v", model, err)
					}
					if client == nil {
						t.Errorf("Expected non-nil client for model %s", model)
					}
				}
			},
		},
		{
			name:        "Groq requires Groq-specific models",
			provider:    "groq",
			description: "Groq should work with Groq-specific models",
			testFunc: func(t *testing.T) {
				models := []string{"llama3-8b-8192", "mixtral-8x7b-32768", "gemma-7b-it"}
				for _, model := range models {
					cfg := &config.Config{
						AIProvider: "groq",
						GroqAPIKey: "gsk-test",
						GroqModel:  model,
					}
					mockDB := &MockDB{shouldError: false}
					client, err := NewClient(cfg, mockDB)
					if err != nil {
						t.Errorf("Failed to create Groq client with model %s: %v", model, err)
					}
					if client == nil {
						t.Errorf("Expected non-nil client for model %s", model)
					}
				}
			},
		},
		{
			name:        "Gemini with default configuration",
			provider:    "gemini",
			description: "Gemini should work with default base URL",
			testFunc: func(t *testing.T) {
				cfg := &config.Config{
					AIProvider:   "gemini",
					GeminiAPIKey: "test-key",
					GeminiModel:  "gemini-2.0-flash",
					// No GeminiBaseURL - should use default
				}
				mockDB := &MockDB{shouldError: false}
				client, err := NewClient(cfg, mockDB)
				if err != nil {
					t.Errorf("Failed to create Gemini client: %v", err)
				}
				if client == nil {
					t.Errorf("Expected non-nil client")
				}
			},
		},
		{
			name:        "Claude with default configuration",
			provider:    "claude",
			description: "Claude should work with default base URL",
			testFunc: func(t *testing.T) {
				cfg := &config.Config{
					AIProvider:   "claude",
					ClaudeAPIKey: "sk-ant-test",
					ClaudeModel:  "claude-3-sonnet",
					// No ClaudeBaseURL - should use default
				}
				mockDB := &MockDB{shouldError: false}
				client, err := NewClient(cfg, mockDB)
				if err != nil {
					t.Errorf("Failed to create Claude client: %v", err)
				}
				if client == nil {
					t.Errorf("Expected non-nil client")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}
