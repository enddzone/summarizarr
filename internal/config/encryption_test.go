package config

import (
	"testing"
)

// TestGenerateKey tests key generation
func TestGenerateKey(t *testing.T) {
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("Expected 64 character key, got %d", len(key))
	}
	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate second key: %v", err)
	}
	if key == key2 {
		t.Error("Generated keys should be different")
	}
}
