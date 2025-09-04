package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// hasSQLCipherLocal mirrors other tests' guard.
func hasSQLCipherLocal(t *testing.T) bool {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Logf("warning: failed to open sqlite3 for SQLCipher check: %v", err)
		return false
	}
	defer func() { _ = db.Close() }()

	var version string
	if err := db.QueryRow("PRAGMA cipher_version").Scan(&version); err != nil || version == "" {
		return false
	}
	return true
}

func TestVerifyWithKey_SucceedsWithCorrectKey(t *testing.T) {
	if !hasSQLCipherLocal(t) {
		t.Skip("SQLCipher not available; skipping VerifyWithKey test")
	}
	key := generateTestEncryptionKey()
	tmp := t.TempDir()
	p := filepath.Join(tmp, "verify_ok.db")

	db, err := NewDB(p, key)
	if err != nil {
		t.Fatalf("failed to create encrypted db: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.VerifyWithKey(key); err != nil {
		t.Fatalf("expected VerifyWithKey to succeed, got error: %v", err)
	}
}

func TestVerifyWithKey_FailsWithWrongKey(t *testing.T) {
	if !hasSQLCipherLocal(t) {
		t.Skip("SQLCipher not available; skipping VerifyWithKey wrong key test")
	}
	correct := generateTestEncryptionKey()
	tmp := t.TempDir()
	p := filepath.Join(tmp, "verify_bad.db")

	db, err := NewDB(p, correct)
	if err != nil {
		t.Fatalf("failed to create encrypted db: %v", err)
	}
	defer func() { _ = db.Close() }()

	wrong := correct[:len(correct)-1] + "0" // tweak last hex nibble
	if err := db.VerifyWithKey(wrong); err == nil {
		t.Fatal("expected VerifyWithKey to fail with wrong key, but it succeeded")
	}
}

func TestNewDB_HeaderIsNotPlainSQLite(t *testing.T) {
	if !hasSQLCipherLocal(t) {
		t.Skip("SQLCipher not available; skipping header test")
	}
	key := generateTestEncryptionKey()
	tmp := t.TempDir()
	p := filepath.Join(tmp, "header_check.db")

	db, err := NewDB(p, key)
	if err != nil {
		t.Fatalf("failed to create encrypted db: %v", err)
	}
	_ = db.Close()

	f, err := os.Open(p)
	if err != nil {
		t.Fatalf("failed to open db file: %v", err)
	}
    defer func() { _ = f.Close() }()

	buf := make([]byte, 16)
	n, err := f.Read(buf)
	if err != nil {
		t.Fatalf("failed to read header: %v", err)
	}
	if n >= 16 && string(buf) == "SQLite format 3\x00" {
		t.Fatalf("database header indicates plaintext SQLite; expected encrypted file")
	}
}
