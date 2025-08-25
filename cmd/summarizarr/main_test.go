package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// writeTestFile writes bytes to path creating parent dir.
func writeTestFile(t *testing.T, path string, data []byte, perm fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, perm); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func Test_backupIfPlaintext_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "test.db")
	writeTestFile(t, db, []byte{}, 0o600)
	if err := backupIfPlaintext(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// File should remain (no backup) because size < 16
	if _, err := os.Stat(db); err != nil {
		t.Fatalf("expected original file to remain: %v", err)
	}
}

func Test_backupIfPlaintext_PlainHeader_Backup(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "plain.db")
	// Create file with SQLite magic header
	hdr := append([]byte("SQLite format 3\x00"), make([]byte, 512)...) // extra bytes
	writeTestFile(t, db, hdr, 0o600)

	// Sidecars
	wal := db + "-wal"
	shm := db + "-shm"
	writeTestFile(t, wal, []byte("wal"), 0o600)
	writeTestFile(t, shm, []byte("shm"), 0o600)

	if err := backupIfPlaintext(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Original should be moved away
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("expected original to be moved, stat err=%v", err)
	}
	// There should be exactly one backup file matching pattern
	matches, err := filepath.Glob(filepath.Join(dir, "plain.db_backup_*.db"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected 1 backup file, got %v, err=%v", matches, err)
	}
	// Sidecars moved best-effort (ignore absence)
	if _, err := os.Stat(wal); !os.IsNotExist(err) {
		t.Fatalf("expected wal moved, err=%v", err)
	}
	if _, err := os.Stat(shm); !os.IsNotExist(err) {
		t.Fatalf("expected shm moved, err=%v", err)
	}
}

func Test_backupIfPlaintext_CorruptedHeader_NoBackup(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "corrupt.db")
	data := append([]byte("XXXXXXXXXXXXXXXZ"), make([]byte, 128)...) // not the magic
	writeTestFile(t, db, data, 0o600)
	if err := backupIfPlaintext(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(db); err != nil {
		t.Fatalf("file should remain: %v", err)
	}
}

func Test_backupIfPlaintext_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "denied.db")
	hdr := append([]byte("SQLite format 3\x00"), make([]byte, 64)...)
	writeTestFile(t, db, hdr, 0o000) // no read perms
	// On some systems, chmod may be relaxed under test; attempt backup and expect error or success
	err := backupIfPlaintext(db)
	// Either we get a permission error, or system allowed read; both are acceptable.
	// If success, original should be moved; if error, original should still exist.
	if err != nil {
		if _, statErr := os.Stat(db); statErr != nil {
			t.Fatalf("expected original to still exist after error, stat err=%v", statErr)
		}
		return
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("expected original moved on success")
	}
}

func Test_backupIfPlaintext_LargeFile_Simulated(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "large.db")
	// Start with header then pad to simulate large file
	buf := append([]byte("SQLite format 3\x00"), make([]byte, 4096*2)...)
	writeTestFile(t, db, buf, 0o600)
	if err := backupIfPlaintext(db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "large.db_backup_*.db"))
	if len(matches) != 1 {
		t.Fatalf("expected 1 backup for large file, got %d", len(matches))
	}
}

func Test_backupIfPlaintext_MissingParentDir(t *testing.T) {
	dir := t.TempDir()
	// Deliberately point to a non-existent subdir
	db := filepath.Join(dir, "missing", "file.db")
	if err := backupIfPlaintext(db); err == nil {
		t.Fatalf("expected error for missing parent directory")
	}
}

func Test_backupIfPlaintext_ConcurrentAccess(t *testing.T) {
	// This test may behave differently on Windows due to file locking semantics.
	if isWindows() {
		t.Skip("skipping on Windows due to rename semantics")
	}
	dir := t.TempDir()
	db := filepath.Join(dir, "concurrent.db")
	hdr := append([]byte("SQLite format 3\x00"), make([]byte, 64)...)
	writeTestFile(t, db, hdr, 0o600)

	// Open file (simulating concurrent reader)
	f, err := os.Open(db)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
    defer func() { _ = f.Close() }()

	// Attempt backup while file is open
	if err := backupIfPlaintext(db); err != nil {
		t.Fatalf("backupIfPlaintext failed during concurrent access: %v", err)
	}
	// Verify moved
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("expected original moved, err=%v", err)
	}
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
