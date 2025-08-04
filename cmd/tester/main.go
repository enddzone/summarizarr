package main

import (
	"context"
	"log"
	"os"
	"time"

	"summarizarr/internal/ai"
	"summarizarr/internal/database"
)

func main() {
	log.Println("Starting summarization tester...")

	db, err := database.NewDB("summarizarr.db")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create a dummy group
	_, err = db.Exec("INSERT OR IGNORE INTO groups (group_id, name) VALUES (?, ?)", "test-group", "Test Group")
	if err != nil {
		log.Fatalf("Failed to insert dummy group: %v", err)
	}
	var groupID int64
	err = db.QueryRow("SELECT id FROM groups WHERE group_id = ?", "test-group").Scan(&groupID)
	if err != nil {
		log.Fatalf("Failed to get dummy group ID: %v", err)
	}

	// Create a dummy user
	_, err = db.Exec("INSERT OR IGNORE INTO users (uuid, number, name) VALUES (?, ?, ?)", "test-user", "+1234567890", "Test User")
	if err != nil {
		log.Fatalf("Failed to insert dummy user: %v", err)
	}
	var userID int64
	err = db.QueryRow("SELECT id FROM users WHERE uuid = ?", "test-user").Scan(&userID)
	if err != nil {
		log.Fatalf("Failed to get dummy user ID: %v", err)
	}

	// Insert some dummy messages
	messages := []string{
		"Hello everyone!",
		"How is the project going?",
		"I think we should focus on the UI next.",
		"Agreed. Let's schedule a meeting for tomorrow to discuss.",
		"Sounds good to me.",
	}

	for _, msg := range messages {
		_, err := db.Exec(`
			INSERT INTO messages (timestamp, message_text, user_id, group_id, is_reaction)
			VALUES (?, ?, ?, ?, 0)
		`, time.Now().Unix(), msg, userID, groupID)
		if err != nil {
			log.Fatalf("Failed to insert dummy message: %v", err)
		}
	}

	log.Println("Dummy data inserted.")

	aiModel := os.Getenv("OPENAI_MODEL")
	if aiModel == "" {
		aiModel = "gpt-4o"
	}

	aiClient := ai.NewClient(os.Getenv("OPENAI_API_KEY"), aiModel)

	// Get the messages for summarization
	messagesForSummary, err := db.GetMessagesForSummarization(groupID, 0, time.Now().Unix())
	if err != nil {
		log.Fatalf("Failed to get messages for summarization: %v", err)
	}

	// Summarize the messages
	summary, err := aiClient.Summarize(context.Background(), messagesForSummary)
	if err != nil {
		log.Fatalf("Failed to summarize messages: %v", err)
	}

	log.Printf("Generated Summary:\n%s\n", summary)

	// Save the summary
	if err := db.SaveSummary(groupID, summary, 0, time.Now().Unix()); err != nil {
		log.Fatalf("Failed to save summary: %v", err)
	}

	log.Println("Tester finished.")
}