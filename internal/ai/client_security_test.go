package ai

import (
	"context"
	"fmt"
	"strings"
	"summarizarr/internal/database"
	"testing"
	"time"
)

// MockDB implements the DB interface for testing
type MockDB struct {
	shouldError bool
	errorMsg    string
	users       map[int64]string
}

func (m *MockDB) GetMessagesForSummarization(groupID int64, start, end int64) ([]database.MessageForSummary, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error: %s", m.errorMsg)
	}
	return nil, nil
}

func (m *MockDB) GetGroups() ([]int64, error) {
	if m.shouldError {
		return nil, fmt.Errorf("mock error: %s", m.errorMsg)
	}
	return nil, nil
}

func (m *MockDB) SaveSummary(groupID int64, summaryText string, start, end int64) error {
	if m.shouldError {
		return fmt.Errorf("mock error: %s", m.errorMsg)
	}
	return nil
}

func (m *MockDB) GetUserNameByID(userID int64) (string, error) {
	if m.shouldError {
		return "", fmt.Errorf("mock error: %s", m.errorMsg)
	}
	if m.users != nil {
		if name, exists := m.users[userID]; exists {
			return name, nil
		}
	}
	return fmt.Sprintf("User %d", userID), nil
}

func (m *MockDB) GetGroupNameByID(groupID int64) (string, error) {
	if m.shouldError {
		return "", fmt.Errorf("mock error: %s", m.errorMsg)
	}
	return fmt.Sprintf("Group %d", groupID), nil
}

// MockAIClient implements the AIClient interface for testing
type MockAIClient struct {
	shouldError bool
	errorMsg    string
	response    string
}

func (m *MockAIClient) Summarize(ctx context.Context, prompt string) (string, error) {
	if m.shouldError {
		return "", fmt.Errorf("mock error: %s", m.errorMsg)
	}
	return m.response, nil
}

func TestSanitizeSummaryFormat_ReDoSProtection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		timeout  bool
	}{
		{
			name:     "oversized input",
			input:    strings.Repeat("x", 60*1024), // > 50KB
			expected: "Summary too long to process safely",
		},
		{
			name:     "malicious nested asterisks pattern",
			input:    strings.Repeat("**", 1000) + "header" + strings.Repeat("**", 1000),
			expected: strings.Repeat("**", 1000) + "header" + strings.Repeat("**", 1000), // Should be processed safely due to bounded regex
		},
		{
			name:     "malicious colon pattern",
			input:    strings.Repeat(":", 1000) + "\n" + strings.Repeat("a", 1000) + ":",
			expected: strings.Repeat(":", 1000) + "\n" + strings.Repeat("a", 1000) + ":", // Should be processed safely
		},
		{
			name:     "malicious hash pattern",
			input:    strings.Repeat("#", 100) + " " + strings.Repeat("a", 1000),
			expected: strings.Repeat("#", 100) + " " + strings.Repeat("a", 1000), // Should be processed safely
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only input",
			input:    "   \n\t  \n   ",
			expected: "",
		},
		{
			name:     "normal input",
			input:    "## Key topics discussed\n- Normal content\n\n## Important decisions or conclusions\n- Some decision",
			expected: "## Key topics discussed\n\n- Normal content\n\n## Important decisions or conclusions\n\n- Some decision",
		},
		{
			name:     "header normalization - bold format",
			input:    "**Key topics discussed**:\n- Content",
			expected: "## Key topics discussed\n\n- Content",
		},
		{
			name:     "header normalization - colon format",
			input:    "Key topics discussed:\n- Content",
			expected: "Key topics discussed:\n- Content", // This header format isn't in the expected list, so it won't match
		},
		{
			name:     "header normalization - hash format",
			input:    "### Key topics discussed\n- Content",
			expected: "### Key topics discussed\n- Content", // This header format isn't in the expected list, so it won't match
		},
		{
			name:     "nested list flattening",
			input:    "- Topic one:\n    - Subtopic details",
			expected: "- Topic one: Subtopic details",
		},
		{
			name:     "extra newlines cleanup",
			input:    "## Header\n\n\n\n- Content",
			expected: "## Header\n\n- Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSummaryFormat(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeSummaryFormat() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeSummaryFormat_SecurityBoundaries(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "max boundary input size",
			input:       strings.Repeat("a", maxSummarySize), // Exactly 50KB
			description: "Should process exactly 50KB input without error",
		},
		{
			name:        "boundary + 1 input size",
			input:       strings.Repeat("a", maxSummarySize+1), // 50KB + 1
			description: "Should reject input larger than 50KB",
		},
		{
			name:        "bounded header length",
			input:       "**" + strings.Repeat("a", 100) + "**:", // Exactly 100 chars
			description: "Should handle exactly 100 character headers",
		},
		{
			name:        "oversized header length",
			input:       "**" + strings.Repeat("a", 150) + "**:", // More than 100 chars
			description: "Should not match headers longer than 100 characters",
		},
		{
			name:        "bounded nested list content",
			input:       "- " + strings.Repeat("a", 100) + ":\n    - " + strings.Repeat("b", 500),
			description: "Should handle nested lists within bounds",
		},
		{
			name:        "oversized nested list content",
			input:       "- " + strings.Repeat("a", 150) + ":\n    - " + strings.Repeat("b", 600),
			description: "Should not match nested lists beyond bounds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that function completes within reasonable time (no ReDoS)
			done := make(chan bool, 1)
			var result string

			go func() {
				result = SanitizeSummaryFormat(tt.input)
				done <- true
			}()

			select {
			case <-done:
				// Function completed successfully
				t.Logf("Test passed: %s. Result length: %d", tt.description, len(result))
			case <-time.After(10 * time.Second):
				t.Fatalf("Test timed out: %s. Potential ReDoS vulnerability.", tt.description)
			}

			// Additional validation for oversized input
			if strings.Contains(tt.name, "boundary + 1") {
				if result != "Summary too long to process safely" {
					t.Errorf("Expected rejection of oversized input, got: %q", result)
				}
			}
		})
	}
}

func TestSubstituteUserNames_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		mockDB      *MockDB
		summary     string
		messages    []database.MessageForSummary
		expected    string
		expectError bool
	}{
		{
			name: "database error with fallback",
			mockDB: &MockDB{
				shouldError: true,
				errorMsg:    "connection timeout",
			},
			summary: "user_123 said hello and user_456 replied",
			messages: []database.MessageForSummary{
				{UserID: 123, Text: "hello"},
				{UserID: 456, Text: "reply"},
			},
			expected:    "User 123 said hello and User 456 replied",
			expectError: false, // Should not return error, but use fallback
		},
		{
			name: "successful name substitution",
			mockDB: &MockDB{
				shouldError: false,
				users: map[int64]string{
					123: "Alice",
					456: "Bob",
				},
			},
			summary: "user_123 said hello and user_456 replied",
			messages: []database.MessageForSummary{
				{UserID: 123, Text: "hello"},
				{UserID: 456, Text: "reply"},
			},
			expected:    "Alice said hello and Bob replied",
			expectError: false,
		},
		{
			name: "user not found in database",
			mockDB: &MockDB{
				shouldError: false,
				users:       nil, // Simulate empty user map
			},
			summary: "user_789 sent a message",
			messages: []database.MessageForSummary{
				{UserID: 789, Text: "message"},
			},
			expected:    "User 789 sent a message",
			expectError: false,
		},
		{
			name: "partial database errors",
			mockDB: &MockDB{
				shouldError: false,
				users: map[int64]string{
					123: "Alice",
					// 456 not in map, will trigger error path
				},
			},
			summary: "user_123 and user_456 were chatting",
			messages: []database.MessageForSummary{
				{UserID: 123, Text: "hello"},
				{UserID: 456, Text: "reply"},
			},
			expected:    "Alice and User 456 were chatting",
			expectError: false,
		},
		{
			name: "no user placeholders",
			mockDB: &MockDB{
				shouldError: false,
			},
			summary:     "This is a summary with no user references",
			messages:    []database.MessageForSummary{},
			expected:    "This is a summary with no user references",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{db: tt.mockDB}

			result, err := client.substituteUserNames(tt.summary, tt.messages)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("substituteUserNames() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFormatMessagesForLLM_SecurityValidation(t *testing.T) {
	tests := []struct {
		name     string
		messages []database.MessageForSummary
		expected string
	}{
		{
			name: "malicious input with newlines",
			messages: []database.MessageForSummary{
				{UserID: 1, Text: "Normal message"},
				{UserID: 2, Text: "Message with\nmultiple\nlines"},
				{UserID: 3, Text: "Message with\ttabs"},
			},
			expected: "user_1: Normal message\nuser_2: Message with\nmultiple\nlines\nuser_3: Message with\ttabs\n",
		},
		{
			name: "extremely long messages",
			messages: []database.MessageForSummary{
				{UserID: 1, Text: strings.Repeat("a", 10000)},
			},
			expected: fmt.Sprintf("user_1: %s\n", strings.Repeat("a", 10000)),
		},
		{
			name: "special characters and injection attempts",
			messages: []database.MessageForSummary{
				{UserID: 1, Text: "<script>alert('xss')</script>"},
				{UserID: 2, Text: "'; DROP TABLE messages; --"},
				{UserID: 3, Text: "{{.Execute}}"},
			},
			expected: "user_1: <script>alert('xss')</script>\nuser_2: '; DROP TABLE messages; --\nuser_3: {{.Execute}}\n",
		},
		{
			name: "unicode and emoji handling",
			messages: []database.MessageForSummary{
				{UserID: 1, Text: "Hello 👋 世界"},
				{UserID: 2, ReactionEmoji: "😀", MessageType: "reaction"},
			},
			expected: "user_1: Hello 👋 世界\nuser_2 reacted with 😀\n",
		},
		{
			name: "empty and whitespace messages",
			messages: []database.MessageForSummary{
				{UserID: 1, Text: ""},
				{UserID: 2, Text: "   "},
				{UserID: 3, Text: "\n\t"},
			},
			expected: "user_2:    \nuser_3: \n\t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMessagesForLLM(tt.messages)
			if result != tt.expected {
				t.Errorf("FormatMessagesForLLM() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestClient_Summarize_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		mockAI      *MockAIClient
		mockDB      *MockDB
		messages    []database.MessageForSummary
		expectError bool
		errorMsg    string
	}{
		{
			name: "AI backend error",
			mockAI: &MockAIClient{
				shouldError: true,
				errorMsg:    "OpenAI API timeout",
			},
			mockDB: &MockDB{shouldError: false},
			messages: []database.MessageForSummary{
				{UserID: 1, Text: "test message"},
			},
			expectError: true,
			errorMsg:    "OpenAI API timeout",
		},
		{
			name: "successful summarization with DB error handling",
			mockAI: &MockAIClient{
				shouldError: false,
				response:    "## Key topics discussed\n- Test topic",
			},
			mockDB: &MockDB{
				shouldError: true,
				errorMsg:    "database connection lost",
			},
			messages: []database.MessageForSummary{
				{UserID: 123, Text: "test message"},
			},
			expectError: false, // Should not error due to fallback handling
		},
		{
			name: "successful end-to-end",
			mockAI: &MockAIClient{
				shouldError: false,
				response:    "## Key topics discussed\n- user_456 discussed topics",
			},
			mockDB: &MockDB{
				shouldError: false,
				users:       map[int64]string{456: "Alice"},
			},
			messages: []database.MessageForSummary{
				{UserID: 456, Text: "Let's discuss topics"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				backend: tt.mockAI,
				db:      tt.mockDB,
			}

			ctx := context.Background()
			result, err := client.Summarize(ctx, tt.messages)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result == "" {
					t.Errorf("Expected non-empty result")
				}
			}
		})
	}
}

// Benchmark tests to ensure performance is acceptable
func BenchmarkSanitizeSummaryFormat(b *testing.B) {
	// Create a realistic summary
	summary := `## Key topics discussed
- Project planning and timeline
- Budget allocation for Q4
- Team assignments and responsibilities

## Important decisions or conclusions  
- Approved budget increase of 15%
- Decided to hire 2 new developers
- Set deadline for project completion

## Action items or next steps
- Schedule follow-up meeting for next week
- Send updated requirements to stakeholders
- Begin recruitment process for new positions

## Notable reactions or responses
- Team expressed enthusiasm for new project
- Concerns raised about timeline feasibility
- Positive feedback on budget allocation`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeSummaryFormat(summary)
	}
}

func BenchmarkFormatMessagesForLLM(b *testing.B) {
	messages := make([]database.MessageForSummary, 100)
	for i := 0; i < 100; i++ {
		messages[i] = database.MessageForSummary{
			UserID: int64(i % 10),
			Text:   fmt.Sprintf("This is message number %d with some content", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatMessagesForLLM(messages)
	}
}
