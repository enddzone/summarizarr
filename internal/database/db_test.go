package database

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestGetSummaries(t *testing.T) {
	// Create a temporary test database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("Failed to close test database: %v", err)
		}
	}()

	// Create schema (full schema from schema.sql)
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE,
		number TEXT,
		name TEXT
	);

	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id TEXT UNIQUE,
		name TEXT
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER,
		server_received_timestamp INTEGER,
		server_delivered_timestamp INTEGER,
		message_text TEXT,
		message_type TEXT DEFAULT 'message',
		
		-- Quote fields
		quote_id INTEGER,
		quote_author_uuid TEXT,
		quote_text TEXT,
		
		-- Reaction fields
		is_reaction BOOLEAN DEFAULT FALSE,
		reaction_emoji TEXT,
		reaction_target_author_uuid TEXT,
		reaction_target_timestamp INTEGER,
		reaction_is_remove BOOLEAN DEFAULT FALSE,
		
		user_id INTEGER,
		group_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users (id),
		FOREIGN KEY (group_id) REFERENCES groups (id)
	);

	CREATE TABLE IF NOT EXISTS summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		summary_text TEXT,
		start_timestamp INTEGER,
		end_timestamp INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups (id)
	);
	`
	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	db := &DB{DB: testDB}

	// Test empty database
	summaries, err := db.GetSummaries()
	if err != nil {
		t.Fatalf("Failed to get summaries from empty database: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries, got %d", len(summaries))
	}

	// Insert test data with explicit timestamps to ensure ordering
	now := time.Now()
	start := now.Add(-time.Hour).Unix()
	end := now.Unix()

	// Insert first summary
	_, err = testDB.Exec(`
		INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp, created_at) 
		VALUES (?, ?, ?, ?, ?)
	`, 1, "Test summary 1", start, end, now.Add(-time.Minute).Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert first summary: %v", err)
	}

	// Insert second summary (created later)
	_, err = testDB.Exec(`
		INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp, created_at) 
		VALUES (?, ?, ?, ?, ?)
	`, 2, "Test summary 2", start+100, end+100, now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert second summary: %v", err)
	}

	// Get summaries
	summaries, err = db.GetSummaries()
	if err != nil {
		t.Fatalf("Failed to get summaries: %v", err)
	}

	// Verify we got 2 summaries
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Verify data integrity
	for i, summary := range summaries {
		if summary.ID == 0 {
			t.Errorf("Summary %d has zero ID", i)
		}
		if summary.Text == "" {
			t.Errorf("Summary %d has empty text", i)
		}
		if summary.Start == 0 && summary.End != 0 {
			t.Errorf("Summary %d has invalid start timestamp", i)
		}
		if summary.End == 0 {
			t.Errorf("Summary %d has zero end timestamp", i)
		}
		if summary.CreatedAt == "" {
			t.Errorf("Summary %d has empty created_at", i)
		}
	}

	// Verify ordering (should be by created_at DESC)
	if len(summaries) >= 2 {
		// The second summary should have been created later, so it should be first
		if summaries[0].Text != "Test summary 2" {
			t.Errorf("Expected first summary to be 'Test summary 2', got '%s'", summaries[0].Text)
		}
		if summaries[1].Text != "Test summary 1" {
			t.Errorf("Expected second summary to be 'Test summary 1', got '%s'", summaries[1].Text)
		}
	}
}
