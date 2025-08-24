package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"       // Original SQLite driver
	_ "github.com/mattn/go-sqlite3" // SQLCipher-enabled driver
)

// TestMigrateToEncryptedDB tests the main migration function
func TestMigrateToEncryptedDB(t *testing.T) {
	testKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	
	tmpDir := t.TempDir()
	sourceDB := filepath.Join(tmpDir, "source.db")
	destDB := filepath.Join(tmpDir, "encrypted.db")
	
	// Create source database with test data
	if err := createTestSourceDB(sourceDB); err != nil {
		t.Fatalf("Failed to create test source database: %v", err)
	}
	
	// Migrate to encrypted database
	err := migrateToEncryptedDB(sourceDB, destDB, testKey, "", true, false)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}
	
	// Verify migration success
	if err := verifyMigrationResult(destDB, testKey); err != nil {
		t.Fatalf("Migration verification failed: %v", err)
	}
}

// TestMigrateToEncryptedDBWithCorruptSource tests handling of corrupt source database
func TestMigrateToEncryptedDBWithCorruptSource(t *testing.T) {
	testKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	
	tmpDir := t.TempDir()
	sourceDB := filepath.Join(tmpDir, "corrupt.db")
	destDB := filepath.Join(tmpDir, "encrypted.db")
	
	// Create corrupt database (invalid SQLite format)
	if err := os.WriteFile(sourceDB, []byte("not a sqlite database"), 0600); err != nil {
		t.Fatalf("Failed to create corrupt source file: %v", err)
	}
	
	// Migration should fail
	err := migrateToEncryptedDB(sourceDB, destDB, testKey, "", true, false)
	if err == nil {
		t.Error("Expected migration to fail with corrupt source database")
	}
	
	// Destination should not be created
	if _, err := os.Stat(destDB); err == nil {
		t.Error("Expected destination database not to be created on migration failure")
	}
}

// TestMigrateToEncryptedDBWithInvalidKey tests handling of invalid encryption key
func TestMigrateToEncryptedDBWithInvalidKey(t *testing.T) {
	invalidKey := "invalid-key"
	
	tmpDir := t.TempDir()
	sourceDB := filepath.Join(tmpDir, "source.db")
	destDB := filepath.Join(tmpDir, "encrypted.db")
	
	// Create source database with test data
	if err := createTestSourceDB(sourceDB); err != nil {
		t.Fatalf("Failed to create test source database: %v", err)
	}
	
	// Migration should fail with invalid key
	err := migrateToEncryptedDB(sourceDB, destDB, invalidKey, "", true, false)
	if err == nil {
		t.Error("Expected migration to fail with invalid encryption key")
	}
}

// TestGetTables tests table listing functionality
func TestGetTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	// Create test database
	if err := createTestSourceDB(dbPath); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Open database and get tables
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer func() {
		_ = db.Close() // Ignore close errors in helper functions
	}()
	
	tables, err := getTables(db)
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}
	
	// Should have expected tables
	expectedTables := []string{"groups", "messages", "summaries", "users"}
	if len(tables) != len(expectedTables) {
		t.Errorf("Expected %d tables, got %d: %v", len(expectedTables), len(tables), tables)
	}
	
	// Verify table names
	for _, expected := range expectedTables {
		found := false
		for _, table := range tables {
			if table == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected table '%s' not found in: %v", expected, tables)
		}
	}
}

// TestMigrateTableTransactionRollback tests transaction rollback on migration failure
func TestMigrateTableTransactionRollback(t *testing.T) {
	testKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	
	tmpDir := t.TempDir()
	sourceDB := filepath.Join(tmpDir, "source.db")
	destDB := filepath.Join(tmpDir, "encrypted.db")
	
	// Create source database
	if err := createTestSourceDB(sourceDB); err != nil {
		t.Fatalf("Failed to create test source database: %v", err)
	}
	
	// Create destination database with encryption
	destDBConn, err := sql.Open("sqlite3", destDB)
	if err != nil {
		t.Fatalf("Failed to create destination database: %v", err)
	}
	defer func() {
		if err := destDBConn.Close(); err != nil {
			t.Logf("Warning: failed to close destination database: %v", err)
		}
	}()
	
	// Set up encryption
	_, err = destDBConn.Exec(fmt.Sprintf("PRAGMA key = 'x\"%s\"'", testKey))
	if err != nil {
		t.Fatalf("Failed to set encryption key: %v", err)
	}
	
	// Create table but with incompatible schema to force migration failure
	_, err = destDBConn.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, incompatible_column TEXT)")
	if err != nil {
		t.Fatalf("Failed to create incompatible table: %v", err)
	}
	
	// Open source database
	sourceDBConn, err := sql.Open("sqlite", sourceDB)
	if err != nil {
		t.Fatalf("Failed to open source database: %v", err)
	}
	defer func() {
		if err := sourceDBConn.Close(); err != nil {
			t.Logf("Warning: failed to close source database: %v", err)
		}
	}()
	
	// Migration should fail due to schema mismatch
	_, err = migrateTable(sourceDBConn, destDBConn, "users")
	if err == nil {
		t.Error("Expected migration to fail with schema mismatch")
	}
	
	// Verify that partial migration was rolled back
	var count int
	err = destDBConn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query users table: %v", err)
	}
	
	// Should still be 0 (no partial data from failed migration)
	if count != 0 {
		t.Errorf("Expected 0 users after failed migration, got %d", count)
	}
}

// TestVerifyMigration tests migration verification
func TestVerifyMigration(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDB := filepath.Join(tmpDir, "source.db")
	
	// Create test databases
	if err := createTestSourceDB(sourceDB); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	
	// Create identical destination database
	destDB := filepath.Join(tmpDir, "dest.db")
	if err := createTestSourceDB(destDB); err != nil {
		t.Fatalf("Failed to create destination database: %v", err)
	}
	
	// Open databases
	sourceDBConn, err := sql.Open("sqlite", sourceDB)
	if err != nil {
		t.Fatalf("Failed to open source database: %v", err)
	}
	defer func() {
		if err := sourceDBConn.Close(); err != nil {
			t.Logf("Warning: failed to close source database: %v", err)
		}
	}()
	
	destDBConn, err := sql.Open("sqlite", destDB)
	if err != nil {
		t.Fatalf("Failed to open destination database: %v", err)
	}
	defer func() {
		if err := destDBConn.Close(); err != nil {
			t.Logf("Warning: failed to close destination database: %v", err)
		}
	}()
	
	// Get tables
	tables, err := getTables(sourceDBConn)
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}
	
	// Verification should pass
	if err := verifyMigration(sourceDBConn, destDBConn, tables); err != nil {
		t.Errorf("Migration verification should pass: %v", err)
	}
}

// TestVerifyMigrationRowCountMismatch tests verification with row count mismatch
func TestVerifyMigrationRowCountMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDB := filepath.Join(tmpDir, "source.db")
	destDB := filepath.Join(tmpDir, "dest.db")
	
	// Create source database
	if err := createTestSourceDB(sourceDB); err != nil {
		t.Fatalf("Failed to create source database: %v", err)
	}
	
	// Create destination database with different data
	if err := createTestSourceDB(destDB); err != nil {
		t.Fatalf("Failed to create destination database: %v", err)
	}
	
	// Add extra row to destination to create mismatch
	destDBConn, err := sql.Open("sqlite", destDB)
	if err != nil {
		t.Fatalf("Failed to open destination database: %v", err)
	}
	defer func() {
		if err := destDBConn.Close(); err != nil {
			t.Logf("Warning: failed to close destination database: %v", err)
		}
	}()
	
	_, err = destDBConn.Exec("INSERT INTO users (uuid, number, name) VALUES (?, ?, ?)", 
		"extra-uuid", "+9999999999", "Extra User")
	if err != nil {
		t.Fatalf("Failed to insert extra row: %v", err)
	}
	
	// Open source database
	sourceDBConn, err := sql.Open("sqlite", sourceDB)
	if err != nil {
		t.Fatalf("Failed to open source database: %v", err)
	}
	defer func() {
		if err := sourceDBConn.Close(); err != nil {
			t.Logf("Warning: failed to close source database: %v", err)
		}
	}()
	
	// Get tables
	tables, err := getTables(sourceDBConn)
	if err != nil {
		t.Fatalf("Failed to get tables: %v", err)
	}
	
	// Verification should fail
	if err := verifyMigration(sourceDBConn, destDBConn, tables); err == nil {
		t.Error("Expected migration verification to fail with row count mismatch")
	}
}

// Helper functions

// createTestSourceDB creates a test SQLite database with sample data
func createTestSourceDB(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close() // Ignore close errors in helper functions
	}()
	
	// Create schema
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
	
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	
	// Insert test data
	testData := []string{
		"INSERT INTO users (uuid, number, name) VALUES ('test-uuid-1', '+1234567890', 'Test User 1')",
		"INSERT INTO users (uuid, number, name) VALUES ('test-uuid-2', '+0987654321', 'Test User 2')",
		"INSERT INTO groups (group_id, name) VALUES ('test-group-1', 'Test Group 1')",
		"INSERT INTO messages (timestamp, message_text, user_id, group_id) VALUES (1000, 'Test message 1', 1, 1)",
		"INSERT INTO messages (timestamp, message_text, user_id, group_id) VALUES (2000, 'Test message 2', 2, 1)",
		"INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp) VALUES (1, 'Test summary', 1000, 2000)",
	}
	
	for _, query := range testData {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute test data query '%s': %v", query, err)
		}
	}
	
	return nil
}

// verifyMigrationResult verifies that migration was successful
func verifyMigrationResult(dbPath, encryptionKey string) error {
	// Open encrypted database and set key via PRAGMA
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close() // Ignore close errors in helper functions
	}()
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA key = "x'%s'"`, encryptionKey)); err != nil {
		return fmt.Errorf("failed to set encryption key: %v", err)
	}
	
	// Verify we can read data
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return fmt.Errorf("failed to query users: %v", err)
	}
	
	if count == 0 {
		return fmt.Errorf("no users found in migrated database")
	}
	
	// Verify specific data
	var name string
	if err := db.QueryRow("SELECT name FROM users WHERE uuid = ?", "test-uuid-1").Scan(&name); err != nil {
		return fmt.Errorf("failed to query specific user: %v", err)
	}
	
	if name != "Test User 1" {
		return fmt.Errorf("user data mismatch: expected 'Test User 1', got '%s'", name)
	}
	
	return nil
}