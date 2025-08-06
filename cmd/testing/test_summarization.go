package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"summarizarr/internal/ai"
	"summarizarr/internal/database"
)

// TestSummarization tests the complete summarization pipeline
func main() {
	// Set up logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	slog.Info("Testing summarization pipeline...")

	// Initialize database
	db, err := database.NewDB("data/summarizarr.db")
	if err != nil {
		slog.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		slog.Error("OPENAI_API_KEY environment variable is required")
		os.Exit(1)
	}

	// Create AI client with gpt-4o-mini
	aiClient := ai.NewClient(apiKey, "gpt-4o-mini")

	// Get all groups that have messages
	groups, err := db.GetGroups()
	if err != nil {
		slog.Error("Failed to get groups", "error", err)
		os.Exit(1)
	}

	if len(groups) == 0 {
		slog.Info("No groups found. Please run the parse_sample.go script first to populate test data.")
		os.Exit(0)
	}

	ctx := context.Background()

	for _, groupID := range groups {
		slog.Info("Testing summarization for group", "group_id", groupID)

		// Get messages from the last 24 hours (or all messages if less than 24h of data)
		endMs := time.Now().UnixMilli()
		startMs := time.Now().Add(-24 * time.Hour).UnixMilli()

		messages, err := db.GetMessagesForSummarization(groupID, startMs, endMs)
		if err != nil {
			slog.Error("Failed to get messages", "group_id", groupID, "error", err)
			continue
		}

		if len(messages) == 0 {
			slog.Info("No recent messages found for group", "group_id", groupID)
			// Try getting all messages for testing
			messages, err = db.GetMessagesForSummarization(groupID, 0, endMs)
			if err != nil {
				slog.Error("Failed to get all messages", "group_id", groupID, "error", err)
				continue
			}
		}

		if len(messages) == 0 {
			slog.Info("No messages found at all for group", "group_id", groupID)
			continue
		}

		slog.Info("Found messages for summarization", "group_id", groupID, "count", len(messages))

		// Log the messages for context
		for i, msg := range messages {
			slog.Debug("Message", "index", i, "type", msg.MessageType, "user", msg.UserName, "text", msg.Text)
		}

		// Generate summary
		slog.Info("Calling OpenAI API for summarization...", "group_id", groupID)
		summary, err := aiClient.Summarize(ctx, messages)
		if err != nil {
			slog.Error("Failed to generate summary", "group_id", groupID, "error", err)
			continue
		}

		slog.Info("Generated summary", "group_id", groupID, "length", len(summary))
		fmt.Printf("\n=== SUMMARY FOR GROUP %d ===\n%s\n========================\n\n", groupID, summary)

		// Save summary to database
		if err := db.SaveSummary(groupID, summary, startMs, endMs); err != nil {
			slog.Error("Failed to save summary", "group_id", groupID, "error", err)
			continue
		}

		slog.Info("Successfully saved summary to database", "group_id", groupID)
	}

	// Verify summaries were saved by retrieving them
	slog.Info("Retrieving saved summaries...")
	summaries, err := db.GetSummaries()
	if err != nil {
		slog.Error("Failed to get summaries", "error", err)
		os.Exit(1)
	}

	slog.Info("Found summaries in database", "count", len(summaries))
	for _, summary := range summaries {
		fmt.Printf("Summary ID: %d, Group: %d, Created: %s\n", summary.ID, summary.GroupID, summary.CreatedAt)
		fmt.Printf("Text: %s\n\n", summary.Text)
	}

	fmt.Printf("✅ Summarization test completed! Generated %d summaries.\n", len(summaries))
}
