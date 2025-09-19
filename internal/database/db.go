package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"summarizarr/internal/signal"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents a connection to the database.
type DB struct {
	*sql.DB
	DataSourceName string
}

// NewDB creates a new database connection with SQLCipher enforced.
func NewDB(dataSourceName string, encryptionKey string) (*DB, error) {
	// Validate encryption key format
	if err := validateEncryptionKey(encryptionKey); err != nil {
		return nil, err
	}

	var (
		db      *sql.DB
		err     error
		dsn     string
		logPath string
	)

	if dataSourceName == ":memory:" {
		dsn = buildSQLCipherMemoryDSN(encryptionKey)
		logPath = ":memory:"
	} else {
		fsPath := dataSourceName
		if strings.HasPrefix(fsPath, "file:") {
			trimmed := strings.TrimPrefix(fsPath, "file:")
			if i := strings.IndexRune(trimmed, '?'); i >= 0 {
				fsPath = trimmed[:i]
			} else {
				fsPath = trimmed
			}
		}
		if !filepath.IsAbs(fsPath) {
			if abs, absErr := filepath.Abs(fsPath); absErr == nil {
				fsPath = abs
			}
		}

		// Proactively remove stale WAL/SHM before opening; they'll be regenerated if needed.
		_ = os.Remove(fsPath + "-wal")
		_ = os.Remove(fsPath + "-shm")

		dsn = buildSQLCipherFileDSN(fsPath, encryptionKey)
		logPath = fsPath
	}

	slog.Info("Opening encrypted database with SQLCipher", "path", logPath)
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open encrypted database: %w", err)
	}

	// Ensure SQLCipher is available and working on the keyed connection.
	if err := verifySQLCipher(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("SQLCipher verification failed: %w", err)
	}

	// Sanity check that the key can read the schema after initial pragmas.
	if err := verifyKeyUsable(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to open encrypted database with provided key: %w", err)
	}

	// Apply safe, non-breaking PRAGMAs after keying
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		slog.Warn("Failed to enable foreign_keys", "error", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		slog.Warn("Failed to set journal_mode=WAL", "error", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		slog.Warn("Failed to set busy_timeout", "error", err)
	}
	if _, err := db.Exec(`PRAGMA synchronous = NORMAL`); err != nil {
		slog.Warn("Failed to set synchronous=NORMAL", "error", err)
	}

	// Test database connectivity
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool for SQLite/SQLCipher after encryption is verified
	db.SetMaxOpenConns(1)    // SQLite works best with single connection
	db.SetMaxIdleConns(1)    // Keep one idle connection
	db.SetConnMaxLifetime(0) // No max lifetime (persistent connection)

	return &DB{DB: db, DataSourceName: dataSourceName}, nil
}

// verifyKeyUsable runs a minimal query that should succeed after keying and ensures
// the provided encryption key can decrypt the sqlite_master schema.
func verifyKeyUsable(db *sql.DB) error {
	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&cnt); err != nil {
		return err
	}
	return nil
}

// buildSQLCipherFileDSN constructs a DSN that applies the SQLCipher key and
// essential settings before any other pragmas are executed by the driver.
func buildSQLCipherFileDSN(path string, hexKey string) string {
	values := url.Values{}
	values.Set("_cipher", "sqlcipher")
	values.Set("_cipher_compatibility", "4")
	values.Set("_legacy", "4")
	values.Set("_legacy_page_size", "4096")
	values.Set("_plaintext_header_size", "0")
	values.Set("_kdf_iter", "256000")
	values.Set("_busy_timeout", "5000")
	values.Set("_cipher_page_size", "4096")
	values.Set("_cipher_hmac_algorithm", "HMAC_SHA512")
	values.Set("_cipher_kdf_algorithm", "PBKDF2_HMAC_SHA512")
	values.Set("_hmac_use", "on")
	values.Set("_hmac_check", "on")
	values.Set("_key", fmt.Sprintf("x'%s'", strings.ToUpper(hexKey)))

	return fmt.Sprintf("file:%s?%s", path, values.Encode())
}

// buildSQLCipherMemoryDSN returns a DSN for in-memory databases with SQLCipher enabled.
func buildSQLCipherMemoryDSN(hexKey string) string {
	values := url.Values{}
	values.Set("cache", "shared")
	values.Set("_cipher", "sqlcipher")
	values.Set("_cipher_compatibility", "4")
	values.Set("_legacy", "4")
	values.Set("_legacy_page_size", "4096")
	values.Set("_plaintext_header_size", "0")
	values.Set("_kdf_iter", "256000")
	values.Set("_busy_timeout", "5000")
	values.Set("_cipher_page_size", "4096")
	values.Set("_cipher_hmac_algorithm", "HMAC_SHA512")
	values.Set("_cipher_kdf_algorithm", "PBKDF2_HMAC_SHA512")
	values.Set("_hmac_use", "on")
	values.Set("_hmac_check", "on")
	values.Set("_key", fmt.Sprintf("x'%s'", strings.ToUpper(hexKey)))

	return "file::memory:?" + values.Encode()
}

// validateEncryptionKey validates that the encryption key is in the correct format
func validateEncryptionKey(key string) error {
	if len(key) != 64 {
		return fmt.Errorf("encryption key must be 64 hex characters, got %d", len(key))
	}

	for _, c := range key {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return fmt.Errorf("encryption key must be valid hexadecimal")
		}
	}

	return nil
}

// verifySQLCipher checks that SQLCipher is working correctly
func verifySQLCipher(db *sql.DB) error {
	// Try a simple SQLCipher-specific PRAGMA command first
	var version string
	err := db.QueryRow("PRAGMA cipher_version").Scan(&version)
	if err != nil || strings.TrimSpace(version) == "" {
		// Some vanilla SQLite builds silently accept unknown PRAGMAs. Do a strict check:
		// Attempt to set a clearly SQLCipher-only pragma and then re-query version again.
		if _, pragmaErr := db.Exec("PRAGMA cipher_compatibility = 4"); pragmaErr != nil {
			return fmt.Errorf("SQLCipher not available (ensure CGO_ENABLED=1 and SQLCipher library installed): %w", err)
		}
		var v2 string
		if err2 := db.QueryRow("PRAGMA cipher_version").Scan(&v2); err2 != nil || strings.TrimSpace(v2) == "" {
			return fmt.Errorf("SQLCipher verification failed: cipher_version unavailable; database would not be encrypted")
		}
		version = v2
	}

	slog.Info("SQLCipher initialized", "version", strings.TrimSpace(version))
	return nil
}

// NOTE: Key rotation support removed. Rekey operation is no longer available.

// VerifyWithKey attempts to open a fresh connection using the provided key.
func (db *DB) VerifyWithKey(key string) error {
	if err := validateEncryptionKey(key); err != nil {
		return err
	}
	// For in-memory databases, we cannot re-open; perform a simple query on current conn
	if db.DataSourceName == ":memory:" {
		var cnt int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&cnt); err != nil {
			return fmt.Errorf("verification query failed on current connection: %w", err)
		}
		return nil
	}
	// Verify using ATTACH with a key on the existing SQLCipher-enabled connection.
	// This avoids differences in linker/driver flags across build tags.
	escPath := strings.ReplaceAll(db.DataSourceName, "'", "''")
	attachStmt := fmt.Sprintf("ATTACH DATABASE '%s' AS verify KEY \"x'%s'\"", escPath, key)
	if _, err := db.Exec(attachStmt); err != nil {
		return fmt.Errorf("failed to attach database for verification: %w", err)
	}
	defer func() { _, _ = db.Exec("DETACH DATABASE verify") }()

	var cnt int
	if err := db.QueryRow("SELECT COUNT(*) FROM verify.sqlite_master").Scan(&cnt); err != nil {
		return fmt.Errorf("failed to verify database with provided key: %w", err)
	}
	return nil
}

// NOTE: Rotation metadata tables and related helpers have been removed.

// Init creates the database schema.
func (db *DB) Init() error {
	// First, check if we need to create/update the schema
	slog.Info("Initializing database schema")

	schemaBytes, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute each statement individually; go-sqlite3 can surface misleading
	// errors (SQLITE_NOMEM) when fed multi-statement scripts directly.
	// Breaking the schema into discrete statements avoids that issue and
	// provides clearer failure context on individual statements.
	if err := execSQLStatements(db.DB, string(schemaBytes)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	// Check if we need to add missing columns to existing tables
	if err := db.migrateSchema(); err != nil {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}

	// After schema operations, verify that the on-disk database is actually encrypted.
	// If a file-based database exists and starts with the plain SQLite header, fail fast.
	if err := ensureOnDiskEncrypted(db.DataSourceName); err != nil {
		return err
	}

	slog.Info("Database schema initialized successfully")
	return nil
}

// ensureOnDiskEncrypted checks the DB file header to ensure it's not a plaintext SQLite database.
// SQLCipher-encrypted databases do not start with the magic header "SQLite format 3\x00".
func ensureOnDiskEncrypted(dsn string) error {
	// Only applies to file-based databases (ignore in-memory or URIs that are not files)
	if dsn == ":memory:" || dsn == "" {
		return nil
	}
	// Extract file path for DSN like "file:..." or raw path
	path := dsn
	if strings.HasPrefix(dsn, "file:") {
		// Strip URI prefix up to first '?' if present
		p := strings.TrimPrefix(dsn, "file:")
		if i := strings.IndexRune(p, '?'); i >= 0 {
			path = p[:i]
		} else {
			path = p
		}
	}
	// If path is relative, resolve to absolute for logging clarity
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Not created yet, nothing to verify
			return nil
		}
		return fmt.Errorf("failed to stat database file: %w", err)
	}
	if fi.IsDir() {
		return fmt.Errorf("database path points to a directory, not a file: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open database file for verification: %w", err)
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read database header: %w", err)
	}
	if n < 16 {
		// Tiny file; treat as suspicious but not an immediate error
		slog.Warn("Database file too small to verify header; proceeding", "path", path, "size", n)
		return nil
	}
	// Plain SQLite magic header: "SQLite format 3\x00"
	if string(buf) == "SQLite format 3\x00" {
		return fmt.Errorf("database at %s is NOT encrypted (plaintext SQLite). Please move or remove it so Summarizarr can create an encrypted database", path)
	}
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

	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("failed to iterate table info: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close table info rows: %w", err)
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

// execSQLStatements executes a semicolon-delimited script one statement at a time.
// This avoids driver quirks with multi-statement Exec calls and surfaces the
// exact statement that fails.
func execSQLStatements(db *sql.DB, script string) error {
	statements := strings.Split(script, ";")
	for _, stmt := range statements {
		tmp := stmt
		for {
			trimmed := strings.TrimSpace(tmp)
			if trimmed == "" {
				tmp = ""
				break
			}
			if strings.HasPrefix(trimmed, "--") {
				newlineIdx := strings.Index(tmp, "\n")
				if newlineIdx < 0 {
					tmp = ""
				} else {
					tmp = tmp[newlineIdx+1:]
				}
				continue
			}
			break
		}
		trimmed := strings.TrimSpace(tmp)
		if trimmed == "" {
			continue
		}
		trimmed = strings.TrimSuffix(trimmed, ";")
		if trimmed = strings.TrimSpace(trimmed); trimmed == "" {
			continue
		}
		if _, err := db.Exec(trimmed); err != nil {
			return fmt.Errorf("statement failed (%s): %w", trimmed, err)
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

	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("failed to iterate table info for %s: %w", tableName, err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close table info rows for %s: %w", tableName, err)
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
