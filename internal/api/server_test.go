package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestGetSummariesEndpoint(t *testing.T) {
	// Create a temporary test database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

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

	// Insert test groups for foreign key constraint
	_, err = testDB.Exec("INSERT INTO groups (id, group_id, name) VALUES (1, 'test-group-1', 'Test Group 1')")
	if err != nil {
		t.Fatalf("Failed to insert test group: %v", err)
	}
	_, err = testDB.Exec("INSERT INTO groups (id, group_id, name) VALUES (2, 'test-group-2', 'Test Group 2')")
	if err != nil {
		t.Fatalf("Failed to insert test group: %v", err)
	}

	// Insert test data
	now := time.Now()
	start := now.Add(-time.Hour).Unix()
	end := now.Unix()

	_, err = testDB.Exec(`
		INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp, created_at) 
		VALUES (?, ?, ?, ?, ?)
	`, 1, "Test summary 1", start, end, now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp, created_at) 
		VALUES (?, ?, ?, ?, ?)
	`, 2, "Test summary 2", start+100, end+100, now.Add(time.Minute).Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create server
	server := NewServer(":8080", testDB, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/summaries", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleGetSummaries(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Parse JSON response
	var summaries []struct {
		ID        int64     `json:"id"`
		GroupID   int64     `json:"group_id"`
		Text      string    `json:"text"`
		Start     time.Time `json:"start"`
		End       time.Time `json:"end"`
		CreatedAt time.Time `json:"created_at"`
	}

	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify we got 2 summaries
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Verify summaries are ordered by created_at DESC
	if len(summaries) >= 2 {
		if summaries[0].Text != "Test summary 2" {
			t.Errorf("Expected first summary to be 'Test summary 2', got '%s'", summaries[0].Text)
		}
		if summaries[1].Text != "Test summary 1" {
			t.Errorf("Expected second summary to be 'Test summary 1', got '%s'", summaries[1].Text)
		}
		if summaries[0].GroupID != 2 {
			t.Errorf("Expected first summary group_id to be 2, got %d", summaries[0].GroupID)
		}
		if summaries[1].GroupID != 1 {
			t.Errorf("Expected second summary group_id to be 1, got %d", summaries[1].GroupID)
		}
	}

	// Verify timestamps are properly converted
	for i, summary := range summaries {
		if summary.Start.IsZero() {
			t.Errorf("Summary %d has zero start time", i)
		}
		if summary.End.IsZero() {
			t.Errorf("Summary %d has zero end time", i)
		}
		if summary.CreatedAt.IsZero() {
			t.Errorf("Summary %d has zero created_at time", i)
		}
		if summary.ID == 0 {
			t.Errorf("Summary %d has zero ID", i)
		}
	}
}

func TestGetSummariesEmpty(t *testing.T) {
	// Create a temporary test database
	testDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer testDB.Close()

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

	// Create server
	server := NewServer(":8080", testDB, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/summaries", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleGetSummaries(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var summaries []interface{}
	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify empty array
	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries, got %d", len(summaries))
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
