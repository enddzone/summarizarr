package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadEncryptionKeyFromEnv tests loading encryption key from environment variable
func TestLoadEncryptionKeyFromEnv(t *testing.T) {
	testKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	
	// Set test environment variable
	t.Setenv("TEST_ENCRYPTION_KEY", testKey)
	
	config := EncryptionConfig{
		Enabled: true,
		KeyEnv:  "TEST_ENCRYPTION_KEY",
	}
	
	key, err := LoadEncryptionKey(config)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if key != testKey {
		t.Errorf("Expected key '%s', got '%s'", testKey, key)
	}
}

// TestLoadEncryptionKeyFromFile tests loading encryption key from file
func TestLoadEncryptionKeyFromFile(t *testing.T) {
	testKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	
	// Create temporary file with key
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key.txt")
	
	if err := os.WriteFile(keyFile, []byte(testKey), 0600); err != nil {
		t.Fatalf("Failed to write test key file: %v", err)
	}
	
	config := EncryptionConfig{
		Enabled: true,
		KeyFile: keyFile,
	}
	
	key, err := LoadEncryptionKey(config)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if key != testKey {
		t.Errorf("Expected key '%s', got '%s'", testKey, key)
	}
}

// TestLoadEncryptionKeyFileWithWhitespace tests key file with surrounding whitespace
func TestLoadEncryptionKeyFileWithWhitespace(t *testing.T) {
	testKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	keyWithWhitespace := "  \n" + testKey + "\n  "
	
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_key_whitespace.txt")
	
	if err := os.WriteFile(keyFile, []byte(keyWithWhitespace), 0600); err != nil {
		t.Fatalf("Failed to write test key file: %v", err)
	}
	
	config := EncryptionConfig{
		Enabled: true,
		KeyFile: keyFile,
	}
	
	key, err := LoadEncryptionKey(config)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if key != testKey {
		t.Errorf("Expected key '%s', got '%s'", testKey, key)
	}
}

// TestLoadEncryptionKeyPrecedence tests that env variable takes precedence over file
func TestLoadEncryptionKeyPrecedence(t *testing.T) {
	envKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	fileKey := "bbc525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	
	// Set environment variable
	t.Setenv("TEST_ENV_KEY", envKey)
	
	// Create file with different key
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test_file_key.txt")
	
	if err := os.WriteFile(keyFile, []byte(fileKey), 0600); err != nil {
		t.Fatalf("Failed to write test key file: %v", err)
	}
	
	config := EncryptionConfig{
		Enabled: true,
		KeyEnv:  "TEST_ENV_KEY",
		KeyFile: keyFile,
	}
	
	key, err := LoadEncryptionKey(config)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	// Should use environment key, not file key
	if key != envKey {
		t.Errorf("Expected env key '%s', got '%s'", envKey, key)
	}
}

// TestLoadEncryptionKeyDisabled tests behavior when encryption is disabled
func TestLoadEncryptionKeyDisabled(t *testing.T) {
	config := EncryptionConfig{
		Enabled: false,
		KeyEnv:  "SOME_KEY_VAR",
	}
	
	key, err := LoadEncryptionKey(config)
	if err != nil {
		t.Fatalf("Expected no error when encryption disabled, got: %v", err)
	}
	
	if key != "" {
		t.Errorf("Expected empty key when encryption disabled, got '%s'", key)
	}
}

// TestLoadEncryptionKeyNoSource tests error when no key source is available
func TestLoadEncryptionKeyNoSource(t *testing.T) {
	// Ensure env var doesn't exist
	os.Unsetenv("NONEXISTENT_KEY")
	
	config := EncryptionConfig{
		Enabled: true,
		KeyEnv:  "NONEXISTENT_KEY",
		KeyFile: "/nonexistent/file.key",
	}
	
	_, err := LoadEncryptionKey(config)
	if err == nil {
		t.Error("Expected error when no key source available")
	}
	// The actual error varies depending on whether it tries env or file first
	// Just ensure we get some error
}

// TestValidateKey tests key validation with various inputs
func TestValidateKey(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		shouldError bool
		expectError string
	}{
		{
			name:        "Valid 64-character lowercase hex",
			key:         "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9",
			shouldError: false,
		},
		{
			name:        "Valid 64-character uppercase hex",
			key:         "AEB525E0AC9F8ACE668C01E381D0D5A772D004623ABE4EC7A2D9100D986101D9",
			shouldError: false,
		},
		{
			name:        "Valid 64-character mixed case hex",
			key:         "AeB525e0aC9f8aCe668C01e381d0D5a772d004623abE4eC7a2d9100d986101D9",
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
			key:         "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9extra",
			shouldError: true,
			expectError: "encryption key must be 64 hex characters",
		},
		{
			name:        "Invalid hex characters",
			key:         "geb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9",
			shouldError: true,
			expectError: "encryption key must be valid hexadecimal",
		},
		{
			name:        "Empty key",
			key:         "",
			shouldError: true,
			expectError: "encryption key must be 64 hex characters",
		},
		{
			name:        "Key with whitespace gets trimmed",
			key:         "  aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9  ",
			shouldError: false,
		},
		{
			name:        "Key with newlines gets trimmed",
			key:         "\naeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9\n",
			shouldError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateKey(tt.key)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for key '%s' but got none", tt.key)
					return
				}
				if tt.expectError != "" && !contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for key '%s', got: %v", tt.key, err)
					return
				}
				
				// Valid keys should be returned in lowercase
				expectedResult := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
				if result != expectedResult {
					t.Errorf("Expected result '%s', got '%s'", expectedResult, result)
				}
			}
		})
	}
}

// TestGenerateKey tests key generation
func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	
	// Test that generated key is valid
	_, err = validateKey(key)
	if err != nil {
		t.Errorf("Generated key is invalid: %v", err)
	}
	
	// Test that it's 64 characters
	if len(key) != 64 {
		t.Errorf("Expected 64 character key, got %d", len(key))
	}
	
	// Test that multiple calls generate different keys
	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate second key: %v", err)
	}
	
	if key == key2 {
		t.Error("Generated keys should be different")
	}
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