//go:build sqlite_crypt

package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// This integration test verifies that when a plaintext DB is present, it is backed up
// and a new encrypted DB can be initialized successfully with a key.
func Test_Migration_Backup_Then_Encrypted_Init(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sum.db")

	// Create a plaintext DB header file
	if err := os.WriteFile(dbPath, append([]byte("SQLite format 3\x00"), make([]byte, 1024)...), 0o600); err != nil {
		t.Fatalf("write plaintext db: %v", err)
	}

	// Run preflight backup
	if err := backupIfPlaintext(dbPath); err != nil {
		t.Fatalf("backupIfPlaintext: %v", err)
	}

	// Ensure original moved aside
	if _, err := os.Stat(dbPath); err == nil {
		t.Fatalf("expected original db moved away")
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "sum.db_backup_*.db"))
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}

	// Now initialize an encrypted DB using the database package on same path
	// Use a fixed test key (64 hex chars)
	key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Open raw and apply key to create encrypted DB on-disk
	raw, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("sql open: %v", err)
	}
	defer raw.Close()
	if _, err := raw.Exec("PRAGMA key = \"x'" + key + "'\";"); err != nil {
		t.Fatalf("apply key: %v", err)
	}
	if _, err := raw.Exec("CREATE TABLE IF NOT EXISTS t (id INTEGER);"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
}
