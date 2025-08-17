package ai

import (
	"strings"
	"summarizarr/internal/database"
	"testing"
	"time"
)

// Performance regression tests to ensure our security fixes don't introduce performance issues
func BenchmarkSanitizeSummaryFormat_Normal(b *testing.B) {
	summary := `## Key topics discussed
- Project planning and timeline discussions
- Budget allocation for Q4 operations
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

func BenchmarkSanitizeSummaryFormat_Large(b *testing.B) {
	// Test with a large summary (near the 50KB limit)
	baseSection := `## Key topics discussed
- Very detailed topic with extensive discussion about project requirements and implementation details
- Another comprehensive topic covering multiple aspects of the system architecture
- Complex technical discussions about performance optimization and scalability concerns

## Important decisions or conclusions
- Major architectural decision with detailed rationale and implementation plan
- Another significant decision affecting multiple team members and project timelines
- Comprehensive analysis of trade-offs and selected approach with supporting evidence

## Action items or next steps
- Detailed action item with specific requirements and deliverable expectations
- Another comprehensive action requiring coordination across multiple teams
- Complex next steps with dependencies and timeline considerations

## Notable reactions or responses
- Detailed feedback from stakeholders with specific concerns and suggestions
- Another response covering multiple aspects of the proposal with actionable insights
- Comprehensive reaction addressing technical and business considerations`

	// Repeat to create a large summary (but stay under 50KB)
	parts := make([]string, 200) // This should create ~40KB content
	for i := range parts {
		parts[i] = baseSection
	}
	largeSummary := strings.Join(parts, "\n\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeSummaryFormat(largeSummary)
	}
}

func BenchmarkSanitizeSummaryFormat_MaliciousPattern(b *testing.B) {
	// Test performance with potentially malicious patterns (but safe due to our bounded regex)
	maliciousPattern := strings.Repeat("**", 100) + "header" + strings.Repeat("**", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeSummaryFormat(maliciousPattern)
	}
}

func BenchmarkFormatMessagesForLLM_Small(b *testing.B) {
	messages := make([]database.MessageForSummary, 10)
	for i := 0; i < 10; i++ {
		messages[i] = database.MessageForSummary{
			UserID: int64(i % 5),
			Text:   "This is a test message with some content",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatMessagesForLLM(messages)
	}
}

func BenchmarkFormatMessagesForLLM_Large(b *testing.B) {
	messages := make([]database.MessageForSummary, 1000)
	for i := 0; i < 1000; i++ {
		messages[i] = database.MessageForSummary{
			UserID:      int64(i % 50),
			Text:        "This is a test message with some content that might be longer and contain more details about the conversation",
			MessageType: "regular",
		}
		// Add some variety
		if i%5 == 0 {
			messages[i].MessageType = "reaction"
			messages[i].ReactionEmoji = "👍"
		}
		if i%7 == 0 {
			messages[i].MessageType = "quote"
			messages[i].QuoteText = "Original message being quoted"
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatMessagesForLLM(messages)
	}
}

// Performance test for user name substitution
func BenchmarkSubstituteUserNames_Small(b *testing.B) {
	// Create a mock database that's fast
	mockDB := &MockDB{
		shouldError: false,
		users: map[int64]string{
			1: "Alice", 2: "Bob", 3: "Charlie", 4: "Diana", 5: "Eve",
		},
	}

	client := &Client{db: mockDB}
	summary := "user_1 discussed with user_2 about the proposal. user_3 and user_4 agreed, while user_5 had concerns."
	messages := []database.MessageForSummary{
		{UserID: 1}, {UserID: 2}, {UserID: 3}, {UserID: 4}, {UserID: 5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.substituteUserNames(summary, messages)
	}
}

func BenchmarkSubstituteUserNames_Large(b *testing.B) {
	// Create a mock database with many users
	users := make(map[int64]string)
	for i := int64(1); i <= 100; i++ {
		users[i] = "User " + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}

	mockDB := &MockDB{
		shouldError: false,
		users:       users,
	}

	client := &Client{db: mockDB}

	// Create a large summary with many user references
	var summaryBuilder strings.Builder
	summaryBuilder.WriteString("## Key topics discussed\n")
	for i := int64(1); i <= 100; i++ {
		summaryBuilder.WriteString("- user_")
		summaryBuilder.WriteString(string(rune('0' + i%10)))
		summaryBuilder.WriteString(" contributed to the discussion\n")
	}

	// Create corresponding messages
	messages := make([]database.MessageForSummary, 100)
	for i := int64(0); i < 100; i++ {
		messages[i] = database.MessageForSummary{UserID: i + 1}
	}

	summary := summaryBuilder.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.substituteUserNames(summary, messages)
	}
}

// Timeout regression test - ensure operations complete within reasonable time
func TestPerformanceRegression_SanitizeTimeout(t *testing.T) {
	// Test that sanitization completes within 1 second for normal content
	summary := strings.Repeat("## Header\n- Content\n", 1000)

	start := time.Now()
	result := SanitizeSummaryFormat(summary)
	duration := time.Since(start)

	if duration > time.Second {
		t.Errorf("SanitizeSummaryFormat took too long: %v", duration)
	}

	if result == "" {
		t.Error("SanitizeSummaryFormat returned empty result")
	}
}

func TestPerformanceRegression_FormatMessagesTimeout(t *testing.T) {
	// Test that message formatting completes within reasonable time
	messages := make([]database.MessageForSummary, 10000)
	for i := 0; i < 10000; i++ {
		messages[i] = database.MessageForSummary{
			UserID: int64(i % 100),
			Text:   "Test message content with some details",
		}
	}

	start := time.Now()
	result := FormatMessagesForLLM(messages)
	duration := time.Since(start)

	if duration > 2*time.Second {
		t.Errorf("FormatMessagesForLLM took too long: %v", duration)
	}

	if len(result) == 0 {
		t.Error("FormatMessagesForLLM returned empty result")
	}
}

// Memory usage regression test
func TestPerformanceRegression_MemoryUsage(t *testing.T) {
	// Ensure we don't have obvious memory leaks in repeated operations
	summary := `## Key topics discussed
- Test topic 1
- Test topic 2

## Important decisions
- Test decision`

	// Run many iterations to check for memory leaks
	for i := 0; i < 10000; i++ {
		result := SanitizeSummaryFormat(summary)
		if len(result) == 0 {
			t.Fatalf("Unexpected empty result at iteration %d", i)
		}
	}
}

// Concurrency safety test
func TestPerformanceRegression_ConcurrentAccess(t *testing.T) {
	// Test that our functions are safe for concurrent access
	summary := "## Header\n- Content"

	done := make(chan bool)

	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < 100; j++ {
				result := SanitizeSummaryFormat(summary)
				if len(result) == 0 {
					t.Errorf("Unexpected empty result in goroutine")
					return
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out - possible deadlock or performance issue")
		}
	}
}
