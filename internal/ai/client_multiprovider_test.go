package ai

import (
	"context"
	"strings"
	"summarizarr/internal/config"
	"summarizarr/internal/database"
	"testing"
)

// TestNewClient_MultiProviderSupport tests the creation of AI clients for different providers
func TestNewClient_MultiProviderSupport(t *testing.T) {
	mockDB := &MockDB{shouldError: false}

	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "OpenAI provider configuration",
			cfg: &config.Config{
				AIProvider:    "openai",
				OpenAIAPIKey:  "sk-test-key",
				OpenAIModel:   "gpt-4",
				OpenAIBaseURL: "https://api.openai.com/v1",
			},
			expectError: false,
		},
		{
			name: "Groq provider configuration",
			cfg: &config.Config{
				AIProvider:  "groq",
				GroqAPIKey:  "gsk-test-key",
				GroqModel:   "llama3-8b-8192",
				GroqBaseURL: "https://api.groq.com/openai/v1",
			},
			expectError: false,
		},
		{
			name: "Gemini provider configuration",
			cfg: &config.Config{
				AIProvider:    "gemini",
				GeminiAPIKey:  "test-key",
				GeminiModel:   "gemini-2.0-flash",
				GeminiBaseURL: "http://localhost:8000/hf/v1",
			},
			expectError: false,
		},
		{
			name: "Claude provider configuration",
			cfg: &config.Config{
				AIProvider:    "claude",
				ClaudeAPIKey:  "sk-ant-test-key",
				ClaudeModel:   "claude-3-sonnet",
				ClaudeBaseURL: "http://localhost:8000/openai/v1",
			},
			expectError: false,
		},
		{
			name: "Ollama local provider configuration",
			cfg: &config.Config{
				AIProvider: "local",
				OllamaHost: "http://localhost:11434",
				LocalModel: "llama2",
			},
			expectError: false,
		},
		{
			name: "Unsupported provider",
			cfg: &config.Config{
				AIProvider: "unsupported",
			},
			expectError: true,
			errorMsg:    "unsupported AI provider: unsupported (supported:",
		},
		{
			name: "Empty provider configuration",
			cfg: &config.Config{
				AIProvider: "",
			},
			expectError: true,
			errorMsg:    "AI_PROVIDER must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg, mockDB)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				if client != nil {
					t.Errorf("Expected nil client on error, got %v", client)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected non-nil client")
				}
				if client != nil && client.backend == nil {
					t.Errorf("Expected non-nil backend")
				}
			}
		})
	}
}

// TestMultiProviderConfiguration tests provider-specific configuration parsing
func TestMultiProviderConfiguration(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		expectedProvider string
		expectedModel    string
		expectedBaseURL  string
	}{
		{
			name: "OpenAI configuration with custom base URL",
			envVars: map[string]string{
				"AI_PROVIDER":     "openai",
				"OPENAI_API_KEY":  "sk-test",
				"OPENAI_MODEL":    "gpt-4-turbo",
				"OPENAI_BASE_URL": "https://custom.openai.com/v1",
			},
			expectedProvider: "openai",
			expectedModel:    "gpt-4-turbo",
			expectedBaseURL:  "https://custom.openai.com/v1",
		},
		{
			name: "Groq configuration with default values",
			envVars: map[string]string{
				"AI_PROVIDER":  "groq",
				"GROQ_API_KEY": "gsk-test",
				"GROQ_MODEL":   "mixtral-8x7b-32768",
			},
			expectedProvider: "groq",
			expectedModel:    "mixtral-8x7b-32768",
			expectedBaseURL:  "", // Should use default
		},
		{
			name: "Gemini configuration via proxy",
			envVars: map[string]string{
				"AI_PROVIDER":     "gemini",
				"GEMINI_API_KEY":  "test-key",
				"GEMINI_MODEL":    "gemini-pro",
				"GEMINI_BASE_URL": "http://localhost:8000/hf/v1",
			},
			expectedProvider: "gemini",
			expectedModel:    "gemini-pro",
			expectedBaseURL:  "http://localhost:8000/hf/v1",
		},
		{
			name: "Claude configuration via proxy",
			envVars: map[string]string{
				"AI_PROVIDER":     "claude",
				"CLAUDE_API_KEY":  "sk-ant-test",
				"CLAUDE_MODEL":    "claude-3-opus",
				"CLAUDE_BASE_URL": "http://localhost:8000/openai/v1",
			},
			expectedProvider: "claude",
			expectedModel:    "claude-3-opus",
			expectedBaseURL:  "http://localhost:8000/openai/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create config from environment variables
			cfg := &config.Config{
				AIProvider:    tt.envVars["AI_PROVIDER"],
				OpenAIAPIKey:  tt.envVars["OPENAI_API_KEY"],
				OpenAIModel:   tt.envVars["OPENAI_MODEL"],
				OpenAIBaseURL: tt.envVars["OPENAI_BASE_URL"],
				GroqAPIKey:    tt.envVars["GROQ_API_KEY"],
				GroqModel:     tt.envVars["GROQ_MODEL"],
				GroqBaseURL:   tt.envVars["GROQ_BASE_URL"],
				GeminiAPIKey:  tt.envVars["GEMINI_API_KEY"],
				GeminiModel:   tt.envVars["GEMINI_MODEL"],
				GeminiBaseURL: tt.envVars["GEMINI_BASE_URL"],
				ClaudeAPIKey:  tt.envVars["CLAUDE_API_KEY"],
				ClaudeModel:   tt.envVars["CLAUDE_MODEL"],
				ClaudeBaseURL: tt.envVars["CLAUDE_BASE_URL"],
			}

			mockDB := &MockDB{shouldError: false}
			client, err := NewClient(cfg, mockDB)

			if err != nil {
				t.Errorf("Unexpected error creating client: %v", err)
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil client")
				return
			}

			// Verify the client was created successfully
			if client.backend == nil {
				t.Errorf("Expected non-nil backend")
			}
		})
	}
}

// TestProviderDefaults tests that each provider uses appropriate default values
func TestProviderDefaults(t *testing.T) {
	mockDB := &MockDB{shouldError: false}

	tests := []struct {
		name        string
		provider    string
		cfg         *config.Config
		description string
	}{
		{
			name:     "OpenAI with minimal config",
			provider: "openai",
			cfg: &config.Config{
				AIProvider:   "openai",
				OpenAIAPIKey: "sk-test",
				OpenAIModel:  "gpt-4",
				// No BaseURL specified - should use default
			},
			description: "Should work with OpenAI default base URL",
		},
		{
			name:     "Groq with minimal config",
			provider: "groq",
			cfg: &config.Config{
				AIProvider: "groq",
				GroqAPIKey: "gsk-test",
				GroqModel:  "llama3-8b-8192",
				// No BaseURL specified - should use default
			},
			description: "Should work with Groq default base URL",
		},
		{
			name:     "Gemini with default base URL",
			provider: "gemini",
			cfg: &config.Config{
				AIProvider:   "gemini",
				GeminiAPIKey: "test-key",
				GeminiModel:  "gemini-2.0-flash",
				// No GeminiBaseURL - should use new default
			},
			description: "Should work with Gemini default base URL",
		},
		{
			name:     "Claude with default base URL",
			provider: "claude",
			cfg: &config.Config{
				AIProvider:   "claude",
				ClaudeAPIKey: "sk-ant-test",
				ClaudeModel:  "claude-3-sonnet",
				// No ClaudeBaseURL - should use new default
			},
			description: "Should work with Claude default base URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg, mockDB)

			if err != nil {
				t.Errorf("Unexpected error: %v. %s", err, tt.description)
				return
			}

			if client == nil {
				t.Errorf("Expected non-nil client. %s", tt.description)
				return
			}

			if client.backend == nil {
				t.Errorf("Expected non-nil backend. %s", tt.description)
			}
		})
	}
}

// TestMultiProviderErrorHandling tests error scenarios for each provider
func TestMultiProviderErrorHandling(t *testing.T) {
	mockDB := &MockDB{shouldError: false}

	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "OpenAI missing API key",
			cfg: &config.Config{
				AIProvider:  "openai",
				OpenAIModel: "gpt-4",
				// Missing OpenAIAPIKey
			},
			expectError: true, // Client creation should now fail with validation
			errorMsg:    "OPENAI_API_KEY is required for openai provider",
		},
		{
			name: "Groq missing API key",
			cfg: &config.Config{
				AIProvider: "groq",
				GroqModel:  "llama3-8b-8192",
				// Missing GroqAPIKey
			},
			expectError: true, // Client creation should now fail with validation
			errorMsg:    "GROQ_API_KEY is required for groq provider",
		},
		{
			name: "Gemini missing API key",
			cfg: &config.Config{
				AIProvider:    "gemini",
				GeminiModel:   "gemini-2.0-flash",
				GeminiBaseURL: "http://localhost:8000/hf/v1",
				// Missing GeminiAPIKey
			},
			expectError: true, // Client creation should now fail with validation
			errorMsg:    "GEMINI_API_KEY is required for gemini provider",
		},
		{
			name: "Claude missing API key",
			cfg: &config.Config{
				AIProvider:    "claude",
				ClaudeModel:   "claude-3-sonnet",
				ClaudeBaseURL: "http://localhost:8000/openai/v1",
				// Missing ClaudeAPIKey
			},
			expectError: true, // Client creation should now fail with validation
			errorMsg:    "CLAUDE_API_KEY is required for claude provider",
		},
		{
			name: "Local missing Ollama host",
			cfg: &config.Config{
				AIProvider: "local",
				LocalModel: "llama2",
				// Missing OllamaHost
			},
			expectError: true, // Should now fail due to validation
			errorMsg:    "OLLAMA_HOST is required for local provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.cfg, mockDB)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Errorf("Expected non-nil client")
				}
			}
		})
	}
}

// TestProviderSummarization tests summarization with mock clients for each provider
func TestProviderSummarization(t *testing.T) {
	mockDB := &MockDB{
		shouldError: false,
		users: map[int64]string{
			123: "Alice",
			456: "Bob",
		},
	}

	// Test messages
	messages := []database.MessageForSummary{
		{UserID: 123, Text: "Hello everyone!"},
		{UserID: 456, Text: "Hi Alice, how are you?"},
		{UserID: 123, Text: "I'm doing great, thanks!"},
	}

	// Mock AI response
	mockResponse := `## Key topics discussed
- user_123 greeted the group
- user_456 asked about user_123's wellbeing

## Important decisions or conclusions
- Friendly conversation established

## Action items or next steps
- Continue the conversation

## Notable reactions or responses
- Positive responses from both participants`

	mockAI := &MockAIClient{
		shouldError: false,
		response:    mockResponse,
	}

	client := &Client{
		backend: mockAI,
		db:      mockDB,
	}

	ctx := context.Background()
	summary, err := client.Summarize(ctx, messages)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if summary == "" {
		t.Errorf("Expected non-empty summary")
	}

	// Verify user name substitution occurred
	if !containsString(summary, "Alice") || !containsString(summary, "Bob") {
		t.Errorf("Expected user names to be substituted in summary: %s", summary)
	}

	// Verify user_ID placeholders were replaced
	if containsString(summary, "user_123") || containsString(summary, "user_456") {
		t.Errorf("Expected user placeholders to be replaced: %s", summary)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
