package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"summarizarr/internal/signal"

	_ "modernc.org/sqlite"
)

// DB represents a connection to the database.
type DB struct {
	*sql.DB
}

// NewDB creates a new database connection.
func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// Init creates the database schema.
func (db *DB) Init() error {
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// SaveMessage saves a message to the database.
func (db *DB) SaveMessage(msg *signal.Envelope) error {
	// Extract message content and group info from either DataMessage or SyncMessage
	var messageText string
	var groupInfo *signal.GroupInfo
	var timestamp int64 = msg.Timestamp

	// Check DataMessage first
	if msg.DataMessage != nil && msg.DataMessage.GroupInfo != nil {
		messageText = msg.DataMessage.Message
		groupInfo = msg.DataMessage.GroupInfo
	} else if msg.SyncMessage != nil && msg.SyncMessage.SentMessage != nil && msg.SyncMessage.SentMessage.GroupInfo != nil {
		// Check SyncMessage for sent messages
		messageText = msg.SyncMessage.SentMessage.Message
		groupInfo = msg.SyncMessage.SentMessage.GroupInfo
		if msg.SyncMessage.SentMessage.Timestamp > 0 {
			timestamp = msg.SyncMessage.SentMessage.Timestamp
		}
	} else {
		// Not a group message or no recognizable content, ignore
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	userID, err := db.findOrCreateUser(tx, msg.SourceUUID, msg.SourceNumber, msg.SourceName)
	if err != nil {
		return fmt.Errorf("failed to find or create user: %w", err)
	}

	groupID, err := db.findOrCreateGroup(tx, groupInfo.GroupID, groupInfo.GroupName)
	if err != nil {
		return fmt.Errorf("failed to find or create group: %w", err)
	}

	_, err = tx.Exec(`
		INSERT INTO messages (timestamp, server_received_timestamp, server_delivered_timestamp, message_text, is_reaction, user_id, group_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, timestamp, msg.ServerReceivedTimestamp, msg.ServerDeliveredTimestamp, messageText, 0, userID, groupID)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	return tx.Commit()
}

func (db *DB) findOrCreateUser(tx *sql.Tx, uuid, number, name string) (int64, error) {
	var id int64
	err := tx.QueryRow("SELECT id FROM users WHERE uuid = ?", uuid).Scan(&id)
	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query user: %w", err)
	}

	res, err := tx.Exec("INSERT INTO users (uuid, number, name) VALUES (?, ?, ?)", uuid, number, name)
	if err != nil {
		return 0, fmt.Errorf("failed to insert user: %w", err)
	}

	return res.LastInsertId()
}

// MessageForSummary holds the data needed to generate a summary.
type MessageForSummary struct {
	UserName string
	Text     string
}

// GetMessagesForSummarization retrieves messages for a given group within a time range.
func (db *DB) GetMessagesForSummarization(groupID int64, start, end int64) ([]MessageForSummary, error) {
	rows, err := db.Query(`
		SELECT u.name, m.message_text
		FROM messages m
		JOIN users u ON m.user_id = u.id
		WHERE m.group_id = ? AND m.timestamp BETWEEN ? AND ? AND m.is_reaction = 0
		ORDER BY m.timestamp ASC
	`, groupID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []MessageForSummary
	for rows.Next() {
		var msg MessageForSummary
		if err := rows.Scan(&msg.UserName, &msg.Text); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// GetGroups retrieves all unique group IDs from the database.
func (db *DB) GetGroups() ([]int64, error) {
	rows, err := db.Query("SELECT id FROM groups")
	if err != nil {
		return nil, fmt.Errorf("failed to query groups: %w", err)
	}
	defer rows.Close()

	var groups []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan group: %w", err)
		}
		groups = append(groups, id)
	}

	return groups, nil
}

// SaveSummary saves a summary to the database.
func (db *DB) SaveSummary(groupID int64, summaryText string, start, end int64) error {
	_, err := db.Exec("INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp) VALUES (?, ?, ?, ?)", groupID, summaryText, start, end)
	if err != nil {
		return fmt.Errorf("failed to insert summary: %w", err)
	}
	return nil
}

// Summary represents a summary record from the database.
type Summary struct {
	ID        int64  `json:"id"`
	GroupID   int64  `json:"group_id"`
	Text      string `json:"text"`
	Start     int64  `json:"start_timestamp"`
	End       int64  `json:"end_timestamp"`
	CreatedAt string `json:"created_at"`
}

// GetSummaries retrieves all summaries from the database ordered by creation time.
func (db *DB) GetSummaries() ([]Summary, error) {
	slog.Debug("Executing GetSummaries query")
	rows, err := db.Query("SELECT id, group_id, summary_text, start_timestamp, end_timestamp, created_at FROM summaries ORDER BY created_at DESC")
	if err != nil {
		slog.Error("Failed to execute GetSummaries query", "error", err)
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()

	var summaries []Summary
	rowCount := 0
	for rows.Next() {
		rowCount++
		var s Summary
		if err := rows.Scan(&s.ID, &s.GroupID, &s.Text, &s.Start, &s.End, &s.CreatedAt); err != nil {
			slog.Error("Failed to scan summary row", "error", err, "rowCount", rowCount)
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		slog.Debug("Scanned summary", "id", s.ID, "groupId", s.GroupID, "textLength", len(s.Text))
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating summary rows", "error", err)
		return nil, fmt.Errorf("error iterating summary rows: %w", err)
	}

	slog.Debug("GetSummaries completed", "count", len(summaries), "rowsProcessed", rowCount)
	return summaries, nil
}

func (db *DB) findOrCreateGroup(tx *sql.Tx, groupID, name string) (int64, error) {
	var id int64
	err := tx.QueryRow("SELECT id FROM groups WHERE group_id = ?", groupID).Scan(&id)
	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query group: %w", err)
	}

	res, err := tx.Exec("INSERT INTO groups (group_id, name) VALUES (?, ?)", groupID, name)
	if err != nil {
		return 0, fmt.Errorf("failed to insert group: %w", err)
	}

	return res.LastInsertId()
}
