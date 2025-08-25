package database

import (
	"path/filepath"
	"testing"
	"time"
)

// setupEncryptionMeta creates the minimal encryption metadata tables for tests
func (db *DB) setupEncryptionMeta(t *testing.T) {
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
		t.Fatalf("failed to setup encryption metadata tables: %v", err)
	}
}

func TestRotationMetadata_SuccessAndFailure(t *testing.T) {
	// Use encrypted db on disk (consistent with other tests)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "meta_test.db")
	key := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	db, err := NewDB(dbPath, key)
	if err != nil {
		t.Fatalf("failed to create encrypted db: %v", err)
	}
	defer func() { _ = db.Close() }()

	db.setupEncryptionMeta(t)

	// initial version should be 1
	var v int
	if err := db.QueryRow("SELECT key_version FROM encryption_info WHERE id=1").Scan(&v); err != nil {
		t.Fatalf("failed to read key_version: %v", err)
	}
	if v != 1 {
		t.Fatalf("expected initial key_version=1, got %d", v)
	}

	// record a failure
	if err := db.RecordRotationFailure("/tmp/backup.db", "simulated error"); err != nil {
		t.Fatalf("RecordRotationFailure failed: %v", err)
	}
	// version should remain 1
	if err := db.QueryRow("SELECT key_version FROM encryption_info WHERE id=1").Scan(&v); err != nil {
		t.Fatalf("failed to read key_version after failure: %v", err)
	}
	if v != 1 {
		t.Fatalf("expected key_version to remain 1 after failure, got %d", v)
	}
	// log entry present and verification_ok=false
	var cnt int
	var ok bool
	if err := db.QueryRow("SELECT COUNT(*), MIN(verification_ok) FROM encryption_rotation_log").Scan(&cnt, &ok); err != nil {
		t.Fatalf("failed to query rotation logs: %v", err)
	}
	if cnt < 1 || ok { // MIN should be false
		t.Fatalf("expected at least one failure log with verification_ok=false")
	}

	// record a success
	if err := db.RecordRotationSuccess("/tmp/backup2.db", true, ""); err != nil {
		t.Fatalf("RecordRotationSuccess failed: %v", err)
	}
	// version should be 2 and last_rotated_at recent
	var last int64
	if err := db.QueryRow("SELECT key_version, COALESCE(last_rotated_at,0) FROM encryption_info WHERE id=1").Scan(&v, &last); err != nil {
		t.Fatalf("failed to read encryption_info after success: %v", err)
	}
	if v != 2 {
		t.Fatalf("expected key_version=2 after success, got %d", v)
	}
	if last == 0 || time.Since(time.Unix(last, 0)) > time.Minute {
		t.Fatalf("unexpected last_rotated_at: %d", last)
	}

	// confirm there is at least one success log
	if err := db.QueryRow("SELECT COUNT(*) FROM encryption_rotation_log WHERE verification_ok=1").Scan(&cnt); err != nil {
		t.Fatalf("failed to query success logs: %v", err)
	}
	if cnt < 1 {
		t.Fatalf("expected at least one success log")
	}
}
