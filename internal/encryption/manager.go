package encryption

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles encryption key loading and generation.
//
// Priority:
// 1) Docker secret at /run/secrets/encryption_key (production)
// 2) Local key file next to the database: <data dir>/encryption.key (development)
//   - If missing, it will be generated with 0600 permissions
type Manager struct {
	DataDir    string
	SecretPath string
	KeyFile    string
	DBPath     string
}

// NewManager creates a new Manager based on the database path.
// The key file is placed in the same directory as the database.
func NewManager(databasePath string) *Manager {
	dataDir := filepath.Dir(databasePath)
	if dataDir == "." || dataDir == "" {
		dataDir = "./data"
	}
	return &Manager{
		DataDir:    dataDir,
		SecretPath: "/run/secrets/encryption_key",
		KeyFile:    filepath.Join(dataDir, "encryption.key"),
		DBPath:     databasePath,
	}
}

// LoadOrCreateKey returns a 64-char lowercase hex key, generating a new one if needed.
func (m *Manager) LoadOrCreateKey() (string, error) {
	// 1) Prefer Docker secret if present
	if key, err := readKeyIfExists(m.SecretPath); err != nil {
		return "", fmt.Errorf("failed reading docker secret: %w", err)
	} else if key != "" {
		return key, nil
	}

	// 2) Use local key file next to the database
	if key, err := readKeyIfExists(m.KeyFile); err != nil {
		return "", fmt.Errorf("failed reading local key file: %w", err)
	} else if key != "" {
		return key, nil
	}

	// 3) Generate new key and persist with 0600 perms
	key, err := generateKey()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(m.DataDir, 0755); err != nil {
		return "", fmt.Errorf("failed creating data dir for key: %w", err)
	}
	// Write with owner-only permissions
	if err := os.WriteFile(m.KeyFile, []byte(key+"\n"), 0600); err != nil {
		return "", fmt.Errorf("failed writing key file: %w", err)
	}
	return key, nil
}

func readKeyIfExists(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if fi.IsDir() {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(b))
	if key == "" {
		return "", nil
	}
	if err := validateKey(key); err != nil {
		return "", err
	}
	return strings.ToLower(key), nil
}

// validateKey ensures the key is a valid 32-byte hex string
func validateKey(key string) error {
	key = strings.TrimSpace(key)
	if len(key) != 64 {
		return fmt.Errorf("encryption key must be 64 hex characters (32 bytes), got %d", len(key))
	}
	for _, char := range key {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return fmt.Errorf("encryption key must be valid hexadecimal")
		}
	}
	return nil
}

// generateKey generates a new 32-byte encryption key and returns it as lowercase hex
func generateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return strings.ToLower(hex.EncodeToString(key)), nil
}
