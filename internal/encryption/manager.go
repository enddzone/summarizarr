package encryption

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"summarizarr/internal/database"
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
		slog.Info("Using encryption key from Docker secret", "path", m.SecretPath)
		return key, nil
	}

	// 2) Use local key file next to the database
	if key, err := readKeyIfExists(m.KeyFile); err != nil {
		return "", fmt.Errorf("failed reading local key file: %w", err)
	} else if key != "" {
		slog.Info("Using encryption key from local file", "path", m.KeyFile)
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
	slog.Info("Generated new encryption key", "path", m.KeyFile)
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

// RotationResult contains details about a rotation attempt
type RotationResult struct {
	NewKey         string
	OldKey         string
	BackupPath     string
	RotatedAtUnix  int64
	VerificationOK bool
	Error          string
}

// writeKeyAtomically writes the key to the key file via a temp file and rename
func (m *Manager) writeKeyAtomically(newKey string) error {
	if err := os.MkdirAll(filepath.Dir(m.KeyFile), 0755); err != nil {
		return err
	}
	tmp := m.KeyFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.ToLower(strings.TrimSpace(newKey))+"\n"), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, m.KeyFile)
}

// copyIfExists copies a file if it exists
func copyIfExists(src, dst string) error {
	fi, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.IsDir() {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// backupDatabase creates a simple backup of the database and its -wal/-shm files (if present)
func (m *Manager) backupDatabase(ts string) (string, error) {
	base := filepath.Base(m.DBPath)
	backupDir := filepath.Join(m.DataDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	backupMain := filepath.Join(backupDir, base+".bak."+ts)
	if err := copyIfExists(m.DBPath, backupMain); err != nil {
		return "", err
	}
	// try to back up sidecar files too
	_ = copyIfExists(m.DBPath+"-wal", backupMain+"-wal")
	_ = copyIfExists(m.DBPath+"-shm", backupMain+"-shm")
	return backupMain, nil
}

// RotateKey performs key rotation using SQLCipher PRAGMA rekey with safety measures.
// Requires an open database connection using the current key.
func (m *Manager) RotateKey(ctx context.Context, db *database.DB) (*RotationResult, error) {
	// Load current key from file or secret
	oldKey, err := m.LoadOrCreateKey()
	if err != nil {
		return nil, err
	}

	newKey, err := generateKey()
	if err != nil {
		return nil, err
	}

	// Create backup first
	ts := fmt.Sprintf("%d", time.Now().Unix())
	backupPath, err := m.backupDatabase(ts)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Attempt WAL checkpoint to minimize side files
	_, _ = db.Exec(`PRAGMA wal_checkpoint(FULL)`)

	// Perform rekey
	if err := db.Rekey(newKey); err != nil {
		// Leave backup and key file unchanged
		slog.Error("Rekey failed", "error", err)
		_ = db.RecordRotationFailure(backupPath, err.Error())
		return &RotationResult{NewKey: newKey, OldKey: oldKey, BackupPath: backupPath, RotatedAtUnix: time.Now().Unix(), VerificationOK: false, Error: err.Error()}, err
	}

	// Verify by opening a new connection with the new key
	if err := db.VerifyWithKey(newKey); err != nil {
		slog.Error("Verification after rekey failed", "error", err)
		_ = db.RecordRotationFailure(backupPath, err.Error())
		return &RotationResult{NewKey: newKey, OldKey: oldKey, BackupPath: backupPath, RotatedAtUnix: time.Now().Unix(), VerificationOK: false, Error: err.Error()}, fmt.Errorf("verification failed: %w", err)
	}

	// Update key file atomically
	if err := m.writeKeyAtomically(newKey); err != nil {
		slog.Error("Failed to update key file after successful rekey", "error", err)
		return &RotationResult{NewKey: newKey, OldKey: oldKey, BackupPath: backupPath, RotatedAtUnix: time.Now().Unix(), VerificationOK: true, Error: err.Error()}, err
	}

	// Update metadata and log
	_ = db.RecordRotationSuccess(backupPath, true, "")

	slog.Info("Encryption key rotation completed successfully", "backup", backupPath)
	return &RotationResult{NewKey: newKey, OldKey: oldKey, BackupPath: backupPath, RotatedAtUnix: time.Now().Unix(), VerificationOK: true, Error: ""}, nil
}
