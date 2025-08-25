package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

// createEncryptionMeta creates the minimal encryption metadata tables for tests
func createEncryptionMeta(t *testing.T, db *database.DB) {
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

func TestRotateEncryptionKey_Endpoint(t *testing.T) {
	if !hasSQLCipher(t) {
		t.Skip("SQLCipher not available; skipping rotation API test")
	}

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "api_rotation.db")

	// Use a fixed initial key
	initialKey := "aeb525e0ac9f8ace668c01e381d0d5a772d004623abe4ec7a2d9100d986101d9"
	db, err := database.NewDB(dbPath, initialKey)
	if err != nil {
		t.Fatalf("failed to open encrypted DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	createEncryptionMeta(t, db)

	// Build server using *database.DB variant
	s := NewServerWithAppDB(":0", db, nil)

	// Call handler directly to bypass auth/CSRF middleware
	req := httptest.NewRequest(http.MethodPost, "/api/rotate-encryption-key", strings.NewReader(""))
	w := httptest.NewRecorder()
	s.handleRotateEncryptionKey(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status             string `json:"status"`
		Message            string `json:"message"`
		BackupCreated      bool   `json:"backup_created"`
		VerificationPassed bool   `json:"verification_passed"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if resp.Status != "success" || !resp.VerificationPassed {
		t.Fatalf("unexpected response: %+v", resp)
	}

	// Read key file and ensure we can reopen DB with new key
	keyBytes, err := os.ReadFile(filepath.Join(tmpDir, "encryption.key"))
	if err != nil {
		t.Fatalf("failed to read updated key file: %v", err)
	}
	newKey := strings.TrimSpace(string(keyBytes))
	if len(newKey) < 64 {
		t.Fatalf("unexpected key file contents")
	}
	newKey = newKey[:64]

	fresh, err := database.NewDB(dbPath, newKey)
	if err != nil {
		t.Fatalf("failed to open DB with rotated key: %v", err)
	}
	_ = fresh.Close()
}
