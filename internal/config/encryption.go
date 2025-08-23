package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

type EncryptionConfig struct {
	Enabled bool
	KeyFile string
	KeyEnv  string
}

// LoadEncryptionKey loads encryption key from file (production) or env (development)
func LoadEncryptionKey(config EncryptionConfig) (string, error) {
	if !config.Enabled {
		return "", nil
	}

	// Try environment variable first (development)
	if envKey := os.Getenv(config.KeyEnv); envKey != "" {
		return validateKey(envKey)
	}

	// Try key file (production)
	if config.KeyFile != "" {
		keyBytes, err := os.ReadFile(config.KeyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read encryption key file: %w", err)
		}
		key := strings.TrimSpace(string(keyBytes))
		return validateKey(key)
	}

	return "", fmt.Errorf("no encryption key found in env %s or file %s", config.KeyEnv, config.KeyFile)
}

// validateKey ensures the key is a valid 32-byte hex string
func validateKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if len(key) != 64 {
		return "", fmt.Errorf("encryption key must be 64 hex characters (32 bytes), got %d", len(key))
	}
	
	// Validate hex format
	for _, char := range key {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return "", fmt.Errorf("encryption key must be valid hexadecimal")
		}
	}
	
	return strings.ToLower(key), nil
}

// GenerateKey generates a new 32-byte encryption key
func GenerateKey() (string, error) {
	// Generate 32 random bytes
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	
	// Return as hex string (64 characters)
	return hex.EncodeToString(key), nil
}