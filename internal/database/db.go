package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
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
	// First, check if we need to create/update the schema
	slog.Info("Initializing database schema")

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute the schema - CREATE TABLE IF NOT EXISTS will handle new tables
	// and existing tables will be left unchanged
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Check if we need to add missing columns to existing tables
	if err := db.migrateSchema(); err != nil {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}

	slog.Info("Database schema initialized successfully")
	return nil
}

// migrateSchema adds missing columns to existing tables
func (db *DB) migrateSchema() error {
	// Migrate messages table
	if err := db.migrateMessagesTable(); err != nil {
		return fmt.Errorf("failed to migrate messages table: %w", err)
	}

	// Migrate summaries table for auth integration
	if err := db.migrateAuthTables(); err != nil {
		return fmt.Errorf("failed to migrate auth tables: %w", err)
	}

	return nil
}

// migrateMessagesTable adds missing columns to the messages table
func (db *DB) migrateMessagesTable() error {
	// Check what columns exist in the messages table
	rows, err := db.Query("PRAGMA table_info(messages)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err, "context", "migrateMessagesTable table_info(messages)")
		}
	}()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		existingColumns[name] = true
	}

	// List of columns that should exist in the messages table
	requiredColumns := map[string]string{
		"message_type":       "TEXT DEFAULT 'message'",
		"quote_id":           "INTEGER",
		"quote_author_uuid":  "TEXT",
		"quote_text":         "TEXT",
		"reaction_is_remove": "BOOLEAN DEFAULT FALSE",
	}

	// Add missing columns
	for column, definition := range requiredColumns {
		if !existingColumns[column] {
			alterSQL := fmt.Sprintf("ALTER TABLE messages ADD COLUMN %s %s", column, definition)
			slog.Info("Adding missing column to messages table", "column", column)
			if _, err := db.Exec(alterSQL); err != nil {
				return fmt.Errorf("failed to add column %s: %w", column, err)
			}
		}
	}

	return nil
}

// migrateAuthTables adds user_id columns for authentication integration
func (db *DB) migrateAuthTables() error {
	// Check summaries table
	if err := db.addColumnIfNotExists("summaries", "user_id", "TEXT"); err != nil {
		return fmt.Errorf("failed to add user_id to summaries: %w", err)
	}

	// Check groups table
	if err := db.addColumnIfNotExists("groups", "created_by", "TEXT"); err != nil {
		return fmt.Errorf("failed to add created_by to groups: %w", err)
	}

	// Create indexes for performance if they don't exist
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_summaries_user_id ON summaries(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_groups_created_by ON groups(created_by)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			slog.Warn("Failed to create index", "sql", indexSQL, "error", err)
			// Continue - indexes are optional for functionality
		}
	}

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't already exist
func (db *DB) addColumnIfNotExists(tableName, columnName, columnDef string) error {
	// Check what columns exist in the table
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return fmt.Errorf("failed to get table info for %s: %w", tableName, err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err, "context", fmt.Sprintf("addColumnIfNotExists table_info(%s)", tableName))
		}
	}()

	existingColumns := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("failed to scan table info for %s: %w", tableName, err)
		}
		existingColumns[name] = true
	}

	// Add column if it doesn't exist
	if !existingColumns[columnName] {
		alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnDef)
		slog.Info("Adding missing column", "table", tableName, "column", columnName)
		if _, err := db.Exec(alterSQL); err != nil {
			return fmt.Errorf("failed to add column %s to %s: %w", columnName, tableName, err)
		}
	}

	return nil
}

// SaveMessage saves a message to the database.
func (db *DB) SaveMessage(msg *signal.Envelope) error {
	// Skip receipt messages - we're not interested in delivery/read receipts
	if msg.ReceiptMessage != nil {
		return nil
	}

	// Extract message content and group info from either DataMessage or SyncMessage
	var messageText string
	var groupInfo *signal.GroupInfo
	timestamp := msg.Timestamp
	messageType := "message"
	var quote *signal.Quote
	var reaction *signal.Reaction

	// Check DataMessage first
	if msg.DataMessage != nil && msg.DataMessage.GroupInfo != nil {
		messageText = msg.DataMessage.Message
		groupInfo = msg.DataMessage.GroupInfo
		quote = msg.DataMessage.Quote
		reaction = msg.DataMessage.Reaction

		// Determine message type
		if reaction != nil {
			messageType = "reaction"
		} else if quote != nil {
			messageType = "quote"
		}
	} else if msg.SyncMessage != nil && msg.SyncMessage.SentMessage != nil && msg.SyncMessage.SentMessage.GroupInfo != nil {
		// Check SyncMessage for sent messages
		messageText = msg.SyncMessage.SentMessage.Message
		groupInfo = msg.SyncMessage.SentMessage.GroupInfo
		reaction = msg.SyncMessage.SentMessage.Reaction
		if msg.SyncMessage.SentMessage.Timestamp > 0 {
			timestamp = msg.SyncMessage.SentMessage.Timestamp
		}

		// Determine message type for sync messages
		if reaction != nil {
			messageType = "reaction"
		}
	} else {
		// Not a group message or no recognizable content, ignore
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Error("Failed to rollback transaction", "error", err)
		}
	}()

	userID, err := db.findOrCreateUser(tx, msg.SourceUUID, msg.SourceNumber, msg.SourceName)
	if err != nil {
		return fmt.Errorf("failed to find or create user: %w", err)
	}

	groupID, err := db.findOrCreateGroup(tx, groupInfo.GroupID, groupInfo.GroupName)
	if err != nil {
		return fmt.Errorf("failed to find or create group: %w", err)
	}

	// Prepare values for insertion
	var quoteID, quoteAuthorUUID, quoteText interface{}
	var isReaction bool
	var reactionEmoji, reactionTargetAuthorUUID interface{}
	var reactionTargetTimestamp interface{}
	var reactionIsRemove bool

	// Handle quote data
	if quote != nil {
		quoteID = quote.ID
		quoteAuthorUUID = quote.AuthorUUID
		quoteText = quote.Text
	}

	// Handle reaction data
	if reaction != nil {
		isReaction = true
		reactionEmoji = reaction.Emoji
		reactionTargetAuthorUUID = reaction.TargetAuthorUUID
		reactionTargetTimestamp = reaction.TargetSentTimestamp
		reactionIsRemove = reaction.IsRemove
	}

	_, err = tx.Exec(`
		INSERT INTO messages (
			timestamp, server_received_timestamp, server_delivered_timestamp, 
			message_text, message_type,
			quote_id, quote_author_uuid, quote_text,
			is_reaction, reaction_emoji, reaction_target_author_uuid, 
			reaction_target_timestamp, reaction_is_remove,
			user_id, group_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, timestamp, msg.ServerReceivedTimestamp, msg.ServerDeliveredTimestamp,
		messageText, messageType,
		quoteID, quoteAuthorUUID, quoteText,
		isReaction, reactionEmoji, reactionTargetAuthorUUID,
		reactionTargetTimestamp, reactionIsRemove,
		userID, groupID)
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
	UserID             int64
	GroupID            int64
	UserName           string
	Text               string
	MessageType        string
	QuoteAuthorUUID    string
	QuoteText          string
	ReactionEmoji      string
	ReactionTargetUUID string
}

// GetMessagesForSummarization retrieves messages for a given group within a time range.
func (db *DB) GetMessagesForSummarization(groupID int64, start, end int64) ([]MessageForSummary, error) {
	rows, err := db.Query(`
SELECT 
	m.user_id,
	m.group_id,
	u.name, 
	COALESCE(m.message_text, '') as message_text,
	m.message_type,
	COALESCE(m.quote_author_uuid, '') as quote_author_uuid,
	COALESCE(m.quote_text, '') as quote_text,
	COALESCE(m.reaction_emoji, '') as reaction_emoji,
	COALESCE(m.reaction_target_author_uuid, '') as reaction_target_uuid
FROM messages m
JOIN users u ON m.user_id = u.id
WHERE m.group_id = ? AND m.timestamp BETWEEN ? AND ?
ORDER BY m.timestamp ASC
`, groupID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err, "context", "GetMessagesForSummarization")
		}
	}()

	var messages []MessageForSummary
	for rows.Next() {
		var msg MessageForSummary
		if err := rows.Scan(&msg.UserID, &msg.GroupID, &msg.UserName, &msg.Text, &msg.MessageType,
			&msg.QuoteAuthorUUID, &msg.QuoteText, &msg.ReactionEmoji, &msg.ReactionTargetUUID); err != nil {
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
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err, "context", "GetGroups")
		}
	}()

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
	GroupName string `json:"group_name"`
	Text      string `json:"text"`
	Start     int64  `json:"start_timestamp"`
	End       int64  `json:"end_timestamp"`
	CreatedAt string `json:"created_at"`
}

// GetSummaries retrieves all summaries from the database ordered by creation time.
func (db *DB) GetSummaries() ([]Summary, error) {
	return db.GetSummariesWithFilters("", "", "", "", "")
}

// GetSummariesWithFilters retrieves summaries with optional filtering
func (db *DB) GetSummariesWithFilters(search, groups, startTimeStr, endTimeStr, sort string) ([]Summary, error) {
	slog.Debug("Executing GetSummariesWithFilters query",
		"search", search,
		"groups", groups,
		"start_time", startTimeStr,
		"end_time", endTimeStr,
		"sort", sort)

	// Build the query with optional filters
	query := `SELECT s.id, s.group_id, COALESCE(g.name, 'Group ' || s.group_id) as group_name, s.summary_text, s.start_timestamp, s.end_timestamp, s.created_at 
	          FROM summaries s 
	          LEFT JOIN groups g ON s.group_id = g.id 
	          WHERE 1=1`
	var args []interface{}

	// Add search filter
	if search != "" {
		query += " AND (s.summary_text LIKE ? OR g.name LIKE ?)"
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Add group filter
	if groups != "" {
		groupIDs := strings.Split(groups, ",")
		if len(groupIDs) > 0 {
			placeholders := make([]string, len(groupIDs))
			for i, groupID := range groupIDs {
				placeholders[i] = "?"
				args = append(args, strings.TrimSpace(groupID))
			}
			query += fmt.Sprintf(" AND s.group_id IN (%s)", strings.Join(placeholders, ","))
		}
	}

	// Add time range filters (convert seconds to milliseconds for database)
	// Use start_timestamp/end_timestamp if available, otherwise fall back to created_at
	if startTimeStr != "" {
		// For start time: summary should end after this time (or be created after this time if timestamps are null)
		query += " AND (CASE WHEN s.end_timestamp IS NOT NULL THEN s.end_timestamp >= ? ELSE datetime(s.created_at) >= datetime(?, 'unixepoch') END)"
		// Convert Unix seconds to milliseconds for database comparison
		if startTimeSec, err := strconv.ParseInt(startTimeStr, 10, 64); err == nil {
			args = append(args, startTimeSec*1000, startTimeSec)
		} else {
			args = append(args, startTimeStr, startTimeStr)
		}
	}
	if endTimeStr != "" {
		// For end time: summary should start before this time (or be created before this time if timestamps are null)
		query += " AND (CASE WHEN s.start_timestamp IS NOT NULL THEN s.start_timestamp <= ? ELSE datetime(s.created_at) <= datetime(?, 'unixepoch') END)"
		// Convert Unix seconds to milliseconds for database comparison
		if endTimeSec, err := strconv.ParseInt(endTimeStr, 10, 64); err == nil {
			args = append(args, endTimeSec*1000, endTimeSec)
		} else {
			args = append(args, endTimeStr, endTimeStr)
		}
	}

	// Add ordering based on sort parameter
	orderBy := "s.created_at DESC" // default to newest first
	if sort == "oldest" {
		orderBy = "s.created_at ASC"
	}
	query += " ORDER BY " + orderBy

	slog.Debug("Executing query", "query", query, "args", args)

	rows, err := db.Query(query, args...)
	if err != nil {
		slog.Error("Failed to execute GetSummariesWithFilters query", "error", err)
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Error("Failed to close rows", "error", err, "context", "GetSummariesWithFilters")
		}
	}()

	var summaries []Summary
	rowCount := 0
	for rows.Next() {
		rowCount++
		var s Summary
		if err := rows.Scan(&s.ID, &s.GroupID, &s.GroupName, &s.Text, &s.Start, &s.End, &s.CreatedAt); err != nil {
			slog.Error("Failed to scan summary row", "error", err, "rowCount", rowCount)
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		slog.Debug("Scanned summary", "id", s.ID, "groupId", s.GroupID, "groupName", s.GroupName, "textLength", len(s.Text))
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		slog.Error("Error iterating summary rows", "error", err)
		return nil, fmt.Errorf("error iterating summary rows: %w", err)
	}

	slog.Debug("GetSummariesWithFilters completed", "count", len(summaries), "rowsProcessed", rowCount)
	return summaries, nil
}

// DeleteSummary removes a summary by its ID.
func (db *DB) DeleteSummary(id int64) error {
	res, err := db.Exec("DELETE FROM summaries WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete summary: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Not found is treated as no-op
		slog.Warn("DeleteSummary: no rows affected", "id", id)
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

// GetUserNameByID retrieves the user name by internal user ID.
func (db *DB) GetUserNameByID(userID int64) (string, error) {
	var name string
	err := db.QueryRow("SELECT name FROM users WHERE id = ?", userID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("failed to get user name for ID %d: %w", userID, err)
	}
	return name, nil
}

// GetGroupNameByID retrieves the group name by internal group ID.
func (db *DB) GetGroupNameByID(groupID int64) (string, error) {
	var name string
	err := db.QueryRow("SELECT name FROM groups WHERE id = ?", groupID).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("failed to get group name for ID %d: %w", groupID, err)
	}
	return name, nil
}
