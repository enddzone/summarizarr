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
	defer testDB.Close()

	// Create schema
	schema := `
	CREATE TABLE summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		summary_text TEXT,
		start_timestamp INTEGER,
		end_timestamp INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
