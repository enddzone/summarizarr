package encryption

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"summarizarr/internal/database"
)

// hasSQLCipher checks whether the running environment has SQLCipher available.
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
		return false
	}
	return true
}

func createEncryptionMetaSchema(t *testing.T, db *database.DB) {
	t.Helper()
	schema := `
CREATE TABLE IF NOT EXISTS encryption_info (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    key_version INTEGER NOT NULL DEFAULT 1,
    last_rotated_at INTEGER DEFAULT NULL
);
INSERT OR IGNORE INTO encryption_info (id, key_version, last_rotated_at) VALUES (1, 1, NULL);
CREATE TABLE IF NOT EXISTS encryption_rotation_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_version INTEGER NOT NULL,
    to_version INTEGER NOT NULL,
    rotated_at INTEGER NOT NULL,
    backup_path TEXT,
    verification_ok BOOLEAN NOT NULL DEFAULT 0,
    error TEXT
);
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to setup encryption metadata schema: %v", err)
	}
}

func TestRotateKey_HappyPath(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available; skipping rotation test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_encrypted.db")
	keyPath := filepath.Join(tmpDir, "encryption.key")

	// Initial test key (64 hex chars)
	initialKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	if err := os.WriteFile(keyPath, []byte(initialKey+"\n"), 0600); err != nil {
		t.Fatalf("failed to write initial key file: %v", err)
	}

	// Open encrypted DB with initial key
	db, err := database.NewDB(dbPath, initialKey)
	if err != nil {
		t.Fatalf("failed to create encrypted DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Prepare required metadata tables
	createEncryptionMetaSchema(t, db)

	mgr := NewManager(dbPath)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := mgr.RotateKey(ctx, db)
	if err != nil {
		t.Fatalf("RotateKey failed: %v", err)
	}
	if res == nil || !res.VerificationOK {
		t.Fatalf("RotateKey verification failed: %+v", res)
	}
	if res.NewKey == initialKey {
		t.Fatalf("expected a different new key after rotation")
	}

	// Key file should be updated to new key
	b, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key file: %v", err)
	}
	if stringValue := string(b); len(stringValue) == 0 || stringValue[:64] != res.NewKey {
		t.Fatalf("key file not updated to new key")
	}

	// Verify metadata updated
	var version int
	var last int64
	if err := db.QueryRow("SELECT key_version, COALESCE(last_rotated_at, 0) FROM encryption_info WHERE id=1").Scan(&version, &last); err != nil {
		t.Fatalf("failed to fetch encryption_info: %v", err)
	}
	if version != 2 {
		t.Fatalf("expected key_version=2, got %d", version)
	}
	if last <= 0 {
		t.Fatalf("expected non-zero last_rotated_at")
	}

	// Verify rotation log
	var cnt int
	var ok bool
	if err := db.QueryRow("SELECT COUNT(*), MIN(verification_ok) FROM encryption_rotation_log").Scan(&cnt, &ok); err != nil {
		t.Fatalf("failed to query rotation log: %v", err)
	}
	if cnt < 1 || !ok {
		t.Fatalf("expected at least one successful rotation log entry")
	}

	// Ensure new key can open the DB
	fresh, err := database.NewDB(dbPath, res.NewKey)
	if err != nil {
		t.Fatalf("failed to open DB with new key: %v", err)
	}
	_ = fresh.Close()
}
