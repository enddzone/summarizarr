package database

import (
	"database/sql"
	"fmt"
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
	if msg.SyncMessage == nil || msg.SyncMessage.SentMessage == nil || msg.SyncMessage.SentMessage.GroupInfo == nil {
		return nil // Not a group message, ignore
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

	groupID, err := db.findOrCreateGroup(tx, msg.SyncMessage.SentMessage.GroupInfo.GroupID, msg.SyncMessage.SentMessage.GroupInfo.GroupName)
	if err != nil {
		return fmt.Errorf("failed to find or create group: %w", err)
	}

	var isReaction bool
	var reactionEmoji, reactionTargetAuthorUUID string
	var reactionTargetTimestamp int64
	if msg.SyncMessage.SentMessage.Reaction != nil {
		isReaction = true
		reactionEmoji = msg.SyncMessage.SentMessage.Reaction.Emoji
		reactionTargetAuthorUUID = msg.SyncMessage.SentMessage.Reaction.TargetAuthorUUID
		reactionTargetTimestamp = msg.SyncMessage.SentMessage.Reaction.TargetSentTimestamp
	}

	_, err = tx.Exec(`
		INSERT INTO messages (timestamp, server_received_timestamp, server_delivered_timestamp, message_text, is_reaction, reaction_emoji, reaction_target_author_uuid, reaction_target_timestamp, user_id, group_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.Timestamp, msg.Timestamp, msg.Timestamp, msg.SyncMessage.SentMessage.Message, isReaction, reactionEmoji, reactionTargetAuthorUUID, reactionTargetTimestamp, userID, groupID)
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