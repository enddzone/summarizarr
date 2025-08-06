package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"summarizarr/internal/database"
	"summarizarr/internal/signal"
)

// ParseSampleResponse parses the sample_response.json file and tests our message structures
func main() {
	// Set up logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// Read the sample response file (assumed to be in the project root)
	data, err := os.ReadFile("sample_response.json")
	if err != nil {
		slog.Error("Failed to read sample_response.json", "error", err)
		os.Exit(1)
	}

	// Parse JSON into our EnvelopeWrapper structures
	var wrappers []signal.EnvelopeWrapper
	if err := json.Unmarshal(data, &wrappers); err != nil {
		slog.Error("Failed to unmarshal JSON", "error", err)
		os.Exit(1)
	}

	slog.Info("Successfully parsed envelope wrappers", "count", len(wrappers))

	// Initialize database for testing
	db, err := database.NewDB("test_parse.db")
	if err != nil {
		slog.Error("Failed to create database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// Process each envelope and test our SaveMessage logic
	for i, wrapper := range wrappers {
		envelope := wrapper.Envelope
		if envelope == nil {
			slog.Warn("Envelope is nil, skipping", "index", i)
			continue
		}

		slog.Info("Processing envelope", "index", i, "timestamp", envelope.Timestamp, "account", wrapper.Account)

		// Log the envelope type and details
		if envelope.DataMessage != nil {
			slog.Info("Data message found",
				"text", envelope.DataMessage.Message,
				"has_quote", envelope.DataMessage.Quote != nil,
				"has_reaction", envelope.DataMessage.Reaction != nil)
		}

		if envelope.ReceiptMessage != nil {
			slog.Info("Receipt message found",
				"when", envelope.ReceiptMessage.When,
				"is_delivery", envelope.ReceiptMessage.IsDelivery,
				"is_read", envelope.ReceiptMessage.IsRead)
		}

		// Test our SaveMessage function
		if err := db.SaveMessage(envelope); err != nil {
			slog.Error("Failed to save message", "error", err, "index", i)
		} else {
			slog.Info("Successfully saved message", "index", i)
		}
	}

	// Query back the messages to verify they were stored correctly
	// Use group ID 1 and a large timestamp range to get all messages
	messages, err := db.GetMessagesForSummarization(1, 0, 9999999999999)
	if err != nil {
		slog.Error("Failed to get messages", "error", err)
		os.Exit(1)
	}

	slog.Info("Retrieved messages for verification", "count", len(messages))
	for _, msg := range messages {
		slog.Info("Retrieved message",
			"type", msg.MessageType,
			"user", msg.UserName,
			"text", msg.Text,
			"quote_text", msg.QuoteText,
			"reaction_emoji", msg.ReactionEmoji)
	}

	fmt.Printf("✅ Successfully processed %d envelope wrappers and stored %d messages\n", len(wrappers), len(messages))
}
