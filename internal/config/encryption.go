package config

// Deprecated: Encryption configuration has been removed. Encryption is now mandatory
// and keys are managed automatically by internal/encryption/manager.
// This file remains to avoid breaking imports in tests that may reference
// config.GenerateKey during transition.

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateKey generates a new 32-byte encryption key (hex-encoded).
func GenerateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
