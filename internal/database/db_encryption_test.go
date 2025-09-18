package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"summarizarr/internal/config"
	"summarizarr/internal/signal"
	"testing"
)

// hasSQLCipher checks whether the running environment has SQLCipher available.
// It attempts to read PRAGMA cipher_version; if not available, we skip encryption-specific tests.
func hasSQLCipher(t *testing.T) bool {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Logf("Warning: failed to open sqlite3 for SQLCipher check: %v", err)
		return false
	}
	defer func() { _ = db.Close() }()

	var version string
	if err := db.QueryRow("PRAGMA cipher_version").Scan(&version); err != nil || version == "" {
		// Not available in this environment
		return false
	}
	return true
}

// TestNewDBWithEncryption tests database creation with encryption enabled
func TestNewDBWithEncryption(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available in this environment; skipping encryption test")
	}
	// Generate test encryption key
	testKey := generateTestEncryptionKey()

	// Create temporary file for database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_encrypted.db")

	// Test creating encrypted database
	db, err := NewDB(dbPath, testKey)
	if err != nil {
		t.Fatalf("Failed to create encrypted database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Logf("Warning: failed to close database: %v", err)
		}
	}()

	// Verify SQLCipher is working
	if err := db.initTestSchema(); err != nil {
		t.Fatalf("Failed to initialize encrypted database: %v", err)
	}

	// Test basic database operations
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping encrypted database: %v", err)
	}

	// Test data storage and retrieval
	testMessage := createTestMessage()
	if err := db.SaveMessage(testMessage); err != nil {
		t.Fatalf("Failed to save message to encrypted database: %v", err)
	}

	// Verify data can be read back
	groups, err := db.GetGroups()
	if err != nil {
		t.Fatalf("Failed to get groups from encrypted database: %v", err)
	}

	if len(groups) == 0 {
		t.Error("Expected at least one group after saving message")
	}
}

// TestNewDBWithoutEncryption tests database creation with encryption disabled
func TestNewDBWithoutEncryption(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available in this environment; skipping mandatory encryption test")
	}
	// With mandatory encryption, NewDB requires a key even for :memory:
	testKey := generateTestEncryptionKey()
	db, err := NewDB(":memory:", testKey)
	if err != nil {
		t.Fatalf("Failed to create encrypted in-memory database: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.initTestSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping encrypted in-memory database: %v", err)
	}
}

// TestEncryptionKeyValidation tests various encryption key validation scenarios
func TestEncryptionKeyValidation(t *testing.T) {
	// Some scenarios require SQLCipher to be present; if not, skip those cases conditionally
	sqlcipher := hasSQLCipher(t)
	tests := []struct {
		name        string
		key         string
		shouldError bool
		expectError string
	}{
		{
			name:        "Valid 64-character hex key",
			key:         generateTestEncryptionKey(),
			shouldError: !sqlcipher, // if SQLCipher missing, NewDB will fail the verification
		},
		{
			name:        "Too short key",
			key:         "abc123",
			shouldError: true,
			expectError: "encryption key must be 64 hex characters",
		},
		{
			name:        "Too long key",
			key:         generateTestEncryptionKey() + "extra",
			shouldError: true,
			expectError: "encryption key must be 64 hex characters",
		},
		{
			name:        "Invalid hex characters",
			key:         "xyz525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9",
			shouldError: true,
			expectError: "encryption key must be valid hexadecimal",
		},
		{
			name:        "Empty key",
			key:         "",
			shouldError: true,
			expectError: "encryption key must be 64 hex characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test key in environment
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test_validation.db")

			db, err := NewDB(dbPath, tt.key)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for key '%s' but got none", tt.key)
					if db != nil {
						_ = db.Close()
					}
					return
				}
				if tt.expectError != "" && !contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					// If SQLCipher is unavailable, we expect an error; otherwise, this is a real failure
					if sqlcipher {
						t.Errorf("Expected no error for valid key, got: %v", err)
					}
				}
				if db != nil {
					_ = db.Close()
				}
			}
		})
	}
}

// TestEncryptionEnabledButNoKey tests error handling when encryption is enabled but no key provided
func TestEncryptionEnabledButNoKey(t *testing.T) {
	// NewDB requires a non-empty valid key; empty key should error
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_no_key.db")
	db, err := NewDB(dbPath, "")
	if db != nil {
		_ = db.Close()
	}
	if err == nil {
		t.Fatal("expected error when no encryption key is provided")
	}
}

// TestEncryptedDatabasePersistence tests that encrypted data persists across database connections
func TestEncryptedDatabasePersistence(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available in this environment; skipping encryption persistence test")
	}
	testKey := generateTestEncryptionKey()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_persistence.db")

	// First connection: Create and populate database
	{
		db, err := NewDB(dbPath, testKey)
		if err != nil {
			t.Fatalf("Failed to create encrypted database: %v", err)
		}

		if err := db.initTestSchema(); err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Save test data
		testMessage := createTestMessage()
		if err := db.SaveMessage(testMessage); err != nil {
			t.Fatalf("Failed to save test message: %v", err)
		}

		// Save test summary
		if err := db.SaveSummary(1, "Test persistence summary", 1000, 2000); err != nil {
			t.Fatalf("Failed to save test summary: %v", err)
		}

		_ = db.Close()
	}

	// Second connection: Verify data persists
	{
		db, err := NewDB(dbPath, testKey)
		if err != nil {
			t.Fatalf("Failed to reopen encrypted database: %v", err)
		}
		defer func() {
			if err := db.Close(); err != nil {
				t.Logf("Warning: failed to close database: %v", err)
			}
		}()

		// Verify summaries exist
		summaries, err := db.GetSummaries()
		if err != nil {
			t.Fatalf("Failed to get summaries: %v", err)
		}

		if len(summaries) == 0 {
			t.Error("Expected summaries to persist across connections")
		}

		if summaries[0].Text != "Test persistence summary" {
			t.Errorf("Expected 'Test persistence summary', got '%s'", summaries[0].Text)
		}

		// Verify groups exist
		groups, err := db.GetGroups()
		if err != nil {
			t.Fatalf("Failed to get groups: %v", err)
		}

		if len(groups) == 0 {
			t.Error("Expected groups to persist across connections")
		}
	}
}

// TestEncryptedDatabaseWithWrongKey tests that wrong key fails to decrypt
func TestEncryptedDatabaseWithWrongKey(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available in this environment; skipping wrong-key test")
	}
	// Create database with one key
	correctKey := generateTestEncryptionKey()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_wrong_key.db")

	// Create and populate database with correct key
	{
		db, err := NewDB(dbPath, correctKey)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}

		if err := db.initTestSchema(); err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}

		// Save test data that will need to be decrypted later
		testMessage := createTestMessage()
		if err := db.SaveMessage(testMessage); err != nil {
			t.Fatalf("Failed to save test message: %v", err)
		}

		// Save test summary
		if err := db.SaveSummary(1, "Test summary for wrong key test", 1000, 2000); err != nil {
			t.Fatalf("Failed to save test summary: %v", err)
		}

		_ = db.Close()
	}

	// Try to open with wrong key
	wrongKey, _ := config.GenerateKey() // Generate a different key

	// Debug: Log the keys to verify they're different
	t.Logf("Correct key: %s", correctKey)
	t.Logf("Wrong key: %s", wrongKey)

	db, err := NewDB(dbPath, wrongKey)
	if err == nil {
		// If, for some reason, opening succeeds, ensure we don't leave resources open
		_ = db.Close()
		t.Fatal("Expected error when opening encrypted database with wrong key")
	}
}

// TestEncryptedDatabaseHeader ensures new databases are created with encrypted headers.
func TestEncryptedDatabaseHeader(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available in this environment; skipping header encryption test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "header.db")
	key := generateTestEncryptionKey()

	db, err := NewDB(dbPath, key)
	if err != nil {
		t.Fatalf("failed to create encrypted database: %v", err)
	}

	if err := db.initTestSchema(); err != nil {
		_ = db.Close()
		t.Fatalf("failed to initialize schema: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("failed to close database: %v", err)
	}

	f, err := os.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open database file: %v", err)
	}
	defer func() { _ = f.Close() }()

	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil {
		t.Fatalf("failed to read database header: %v", err)
	}
	if n < 16 {
		t.Fatalf("expected to read 16 bytes from header, got %d", n)
	}

	if string(buf) == "SQLite format 3\x00" {
		t.Fatal("database header is plaintext; expected SQLCipher-encrypted header")
	}
}

// Helper functions for testing

// createTestMessage creates a sample signal message for testing
func createTestMessage() *signal.Envelope {
	return &signal.Envelope{
		Timestamp:    1000,
		SourceUUID:   "test-uuid",
		SourceNumber: "+1234567890",
		SourceName:   "Test User",
		DataMessage: &signal.DataMessage{
			Message: "Test message content",
			GroupInfo: &signal.GroupInfo{
				GroupID:   "test-group-id",
				GroupName: "Test Group",
			},
		},
	}
}

// generateTestEncryptionKey generates a valid encryption key for testing
func generateTestEncryptionKey() string {
	return "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
}

// contains checks if a string contains a substring (helper for error checking)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) ||
		containsInner(s, substr))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
