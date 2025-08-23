package database

import (
	"os"
	"path/filepath"
	"testing"
	"summarizarr/internal/config"
	"summarizarr/internal/signal"
)

// TestNewDBWithEncryption tests database creation with encryption enabled
func TestNewDBWithEncryption(t *testing.T) {
	// Generate test encryption key
	testKey := generateTestEncryptionKey()
	
	// Set environment variable for test
	t.Setenv("TEST_ENCRYPTION_KEY", testKey)
	
	encryptionConfig := config.EncryptionConfig{
		Enabled: true,
		KeyEnv:  "TEST_ENCRYPTION_KEY",
	}
	
	// Create temporary file for database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_encrypted.db")
	
	// Test creating encrypted database
	db, err := NewDB(dbPath, encryptionConfig)
	if err != nil {
		t.Fatalf("Failed to create encrypted database: %v", err)
	}
	defer db.Close()
	
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
	encryptionConfig := config.EncryptionConfig{
		Enabled: false,
	}
	
	// Create in-memory database
	db, err := NewDB(":memory:", encryptionConfig)
	if err != nil {
		t.Fatalf("Failed to create unencrypted database: %v", err)
	}
	defer db.Close()
	
	// Initialize schema
	if err := db.initTestSchema(); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	
	// Test basic operations
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping unencrypted database: %v", err)
	}
}

// TestEncryptionKeyValidation tests various encryption key validation scenarios
func TestEncryptionKeyValidation(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		shouldError bool
		expectError string
	}{
		{
			name:        "Valid 64-character hex key",
			key:         generateTestEncryptionKey(),
			shouldError: false,
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
			t.Setenv("TEST_KEY_VALIDATION", tt.key)
			
			encryptionConfig := config.EncryptionConfig{
				Enabled: true,
				KeyEnv:  "TEST_KEY_VALIDATION",
			}
			
			tmpDir := t.TempDir()
			dbPath := filepath.Join(tmpDir, "test_validation.db")
			
			db, err := NewDB(dbPath, encryptionConfig)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for key '%s' but got none", tt.key)
					if db != nil {
						db.Close()
					}
					return
				}
				if tt.expectError != "" && !contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for valid key, got: %v", err)
				}
				if db != nil {
					db.Close()
				}
			}
		})
	}
}

// TestEncryptionEnabledButNoKey tests error handling when encryption is enabled but no key provided
func TestEncryptionEnabledButNoKey(t *testing.T) {
	// Unset any potential test keys
	os.Unsetenv("TEST_NO_KEY")
	
	encryptionConfig := config.EncryptionConfig{
		Enabled: true,
		KeyEnv:  "TEST_NO_KEY", // This env var doesn't exist
	}
	
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_no_key.db")
	
	db, err := NewDB(dbPath, encryptionConfig)
	if err == nil {
		t.Error("Expected error when encryption enabled but no key provided")
		if db != nil {
			db.Close()
		}
		return
	}
	
	expectedError := "encryption is enabled but no encryption key provided"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got: %v", expectedError, err)
	}
}

// TestEncryptedDatabasePersistence tests that encrypted data persists across database connections
func TestEncryptedDatabasePersistence(t *testing.T) {
	testKey := generateTestEncryptionKey()
	t.Setenv("PERSISTENCE_TEST_KEY", testKey)
	
	encryptionConfig := config.EncryptionConfig{
		Enabled: true,
		KeyEnv:  "PERSISTENCE_TEST_KEY",
	}
	
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_persistence.db")
	
	// First connection: Create and populate database
	{
		db, err := NewDB(dbPath, encryptionConfig)
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
		
		db.Close()
	}
	
	// Second connection: Verify data persists
	{
		db, err := NewDB(dbPath, encryptionConfig)
		if err != nil {
			t.Fatalf("Failed to reopen encrypted database: %v", err)
		}
		defer db.Close()
		
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
	// Create database with one key
	correctKey := generateTestEncryptionKey()
	t.Setenv("CORRECT_KEY", correctKey)
	
	encryptionConfig := config.EncryptionConfig{
		Enabled: true,
		KeyEnv:  "CORRECT_KEY",
	}
	
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_wrong_key.db")
	
	// Create and populate database with correct key
	{
		db, err := NewDB(dbPath, encryptionConfig)
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		
		if err := db.initTestSchema(); err != nil {
			t.Fatalf("Failed to initialize database: %v", err)
		}
		
		db.Close()
	}
	
	// Try to open with wrong key
	wrongKey, _ := config.GenerateKey() // Generate a different key
	t.Setenv("WRONG_KEY", wrongKey)
	
	wrongEncryptionConfig := config.EncryptionConfig{
		Enabled: true,
		KeyEnv:  "WRONG_KEY",
	}
	
	db, err := NewDB(dbPath, wrongEncryptionConfig)
	if err != nil {
		t.Fatalf("Database creation should succeed even with wrong key: %v", err)
	}
	defer db.Close()
	
	// Attempt to read data should fail
	_, err = db.GetSummaries()
	if err == nil {
		t.Error("Expected error when reading encrypted database with wrong key")
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