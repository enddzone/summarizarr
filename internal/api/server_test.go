package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetSummariesEndpoint(t *testing.T) {
	// Create a temporary test database
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("Failed to close test database: %v", err)
		}
	}()

	// Create schema (full schema from schema.sql)
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE,
		number TEXT,
		name TEXT
	);

	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id TEXT UNIQUE,
		name TEXT
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER,
		server_received_timestamp INTEGER,
		server_delivered_timestamp INTEGER,
		message_text TEXT,
		message_type TEXT DEFAULT 'message',
		
		-- Quote fields
		quote_id INTEGER,
		quote_author_uuid TEXT,
		quote_text TEXT,
		
		-- Reaction fields
		is_reaction BOOLEAN DEFAULT FALSE,
		reaction_emoji TEXT,
		reaction_target_author_uuid TEXT,
		reaction_target_timestamp INTEGER,
		reaction_is_remove BOOLEAN DEFAULT FALSE,
		
		user_id INTEGER,
		group_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users (id),
		FOREIGN KEY (group_id) REFERENCES groups (id)
	);

	CREATE TABLE IF NOT EXISTS summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		summary_text TEXT,
		start_timestamp INTEGER,
		end_timestamp INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups (id)
	);
	`
	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Insert test groups for foreign key constraint
	_, err = testDB.Exec("INSERT INTO groups (id, group_id, name) VALUES (1, 'test-group-1', 'Test Group 1')")
	if err != nil {
		t.Fatalf("Failed to insert test group: %v", err)
	}
	_, err = testDB.Exec("INSERT INTO groups (id, group_id, name) VALUES (2, 'test-group-2', 'Test Group 2')")
	if err != nil {
		t.Fatalf("Failed to insert test group: %v", err)
	}

	// Insert test data
	now := time.Now()
	start := now.Add(-time.Hour).Unix()
	end := now.Unix()

	_, err = testDB.Exec(`
		INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp, created_at) 
		VALUES (?, ?, ?, ?, ?)
	`, 1, "Test summary 1", start, end, now.Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	_, err = testDB.Exec(`
		INSERT INTO summaries (group_id, summary_text, start_timestamp, end_timestamp, created_at) 
		VALUES (?, ?, ?, ?, ?)
	`, 2, "Test summary 2", start+100, end+100, now.Add(time.Minute).Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create server
	server := NewServer(":8080", testDB, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/summaries", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleGetSummaries(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Parse JSON response
	var summaries []struct {
		ID        int64     `json:"id"`
		GroupID   int64     `json:"group_id"`
		Text      string    `json:"text"`
		Start     time.Time `json:"start"`
		End       time.Time `json:"end"`
		CreatedAt time.Time `json:"created_at"`
	}

	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify we got 2 summaries
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Verify summaries are ordered by created_at DESC
	if len(summaries) >= 2 {
		if summaries[0].Text != "Test summary 2" {
			t.Errorf("Expected first summary to be 'Test summary 2', got '%s'", summaries[0].Text)
		}
		if summaries[1].Text != "Test summary 1" {
			t.Errorf("Expected second summary to be 'Test summary 1', got '%s'", summaries[1].Text)
		}
		if summaries[0].GroupID != 2 {
			t.Errorf("Expected first summary group_id to be 2, got %d", summaries[0].GroupID)
		}
		if summaries[1].GroupID != 1 {
			t.Errorf("Expected second summary group_id to be 1, got %d", summaries[1].GroupID)
		}
	}

	// Verify timestamps are properly converted
	for i, summary := range summaries {
		if summary.Start.IsZero() {
			t.Errorf("Summary %d has zero start time", i)
		}
		if summary.End.IsZero() {
			t.Errorf("Summary %d has zero end time", i)
		}
		if summary.CreatedAt.IsZero() {
			t.Errorf("Summary %d has zero created_at time", i)
		}
		if summary.ID == 0 {
			t.Errorf("Summary %d has zero ID", i)
		}
	}
}

func TestGetSummariesEmpty(t *testing.T) {
	// Create a temporary test database
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("Failed to close test database: %v", err)
		}
	}()

	// Create schema (full schema from schema.sql)
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE,
		number TEXT,
		name TEXT
	);

	CREATE TABLE IF NOT EXISTS groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id TEXT UNIQUE,
		name TEXT
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp INTEGER,
		server_received_timestamp INTEGER,
		server_delivered_timestamp INTEGER,
		message_text TEXT,
		message_type TEXT DEFAULT 'message',
		
		-- Quote fields
		quote_id INTEGER,
		quote_author_uuid TEXT,
		quote_text TEXT,
		
		-- Reaction fields
		is_reaction BOOLEAN DEFAULT FALSE,
		reaction_emoji TEXT,
		reaction_target_author_uuid TEXT,
		reaction_target_timestamp INTEGER,
		reaction_is_remove BOOLEAN DEFAULT FALSE,
		
		user_id INTEGER,
		group_id INTEGER,
		FOREIGN KEY (user_id) REFERENCES users (id),
		FOREIGN KEY (group_id) REFERENCES groups (id)
	);

	CREATE TABLE IF NOT EXISTS summaries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		group_id INTEGER,
		summary_text TEXT,
		start_timestamp INTEGER,
		end_timestamp INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (group_id) REFERENCES groups (id)
	);
	`
	if _, err := testDB.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Create server
	server := NewServer(":8080", testDB, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/summaries", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleGetSummaries(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse JSON response
	var summaries []interface{}
	if err := json.NewDecoder(w.Body).Decode(&summaries); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify empty array
	if len(summaries) != 0 {
		t.Errorf("Expected 0 summaries, got %d", len(summaries))
	}
}

// QR Code Proxy Tests

func TestHandleSignalQrCode_Success(t *testing.T) {
	// Setup mock Signal CLI server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Handle health check during server initialization
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != "/v1/qrcodelink" {
			t.Errorf("Expected path /v1/qrcodelink, got %s", r.URL.Path)
		}
		deviceName := r.URL.Query().Get("device_name")
		if deviceName != "test-device" {
			t.Errorf("Expected device_name=test-device, got %s", deviceName)
		}

		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake-qr-code-data"))
	}))
	defer mockServer.Close()

	// Set Signal URL to mock server
	originalSignalURL := os.Getenv("SIGNAL_URL")
	defer func() {
		if originalSignalURL != "" {
			_ = os.Setenv("SIGNAL_URL", originalSignalURL)
		} else {
			_ = os.Unsetenv("SIGNAL_URL")
		}
	}()
	_ = os.Setenv("SIGNAL_URL", mockServer.URL[7:]) // Remove http://

	// Create test database
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	// Create server
	server := NewServer(":8080", testDB, nil)

	// Create test request
	req := httptest.NewRequest("GET", "/api/signal/qrcode?device_name=test-device", nil)
	w := httptest.NewRecorder()

	// Call handler
	server.handleSignalQrCode(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Check content type header was copied
	if w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Expected Content-Type: image/png, got %s", w.Header().Get("Content-Type"))
	}

	// Check response body
	if w.Body.String() != "fake-qr-code-data" {
		t.Errorf("Expected response body 'fake-qr-code-data', got '%s'", w.Body.String())
	}
}

func TestHandleSignalQrCode_DefaultDeviceName(t *testing.T) {
	// Setup mock Signal CLI server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Handle health check during server initialization
			w.WriteHeader(http.StatusOK)
			return
		}
		deviceName := r.URL.Query().Get("device_name")
		if deviceName != "summarizarr" {
			t.Errorf("Expected default device_name=summarizarr, got %s", deviceName)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("qr-code"))
	}))
	defer mockServer.Close()

	// Set Signal URL to mock server
	originalSignalURL := os.Getenv("SIGNAL_URL")
	defer func() {
		if originalSignalURL != "" {
			_ = os.Setenv("SIGNAL_URL", originalSignalURL)
		} else {
			_ = os.Unsetenv("SIGNAL_URL")
		}
	}()
	_ = os.Setenv("SIGNAL_URL", mockServer.URL[7:])

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	server := NewServer(":8080", testDB, nil)

	// Request without device_name parameter
	req := httptest.NewRequest("GET", "/api/signal/qrcode", nil)
	w := httptest.NewRecorder()

	server.handleSignalQrCode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleSignalQrCode_InvalidDeviceName(t *testing.T) {
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	// Disable Signal validation for this test to avoid network calls
	server := NewServerWithOptions(":8080", testDB, nil, WithSignalValidation(false))

	testCases := []struct {
		name       string
		deviceName string
	}{
		{"special characters", "device@name"},
		{"spaces", "device%20name"}, // URL-encoded space
		{"too long", strings.Repeat("a", 51)},
		{"sql injection", "device%27%3B%20DROP%20TABLE%20users%3B%20--"}, // URL-encoded
		{"url injection", "device%26param%3Dvalue"},                      // URL-encoded
		{"path traversal", "..%2Fdevice"},                                // URL-encoded
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("/api/signal/qrcode?device_name=%s", tc.deviceName)
			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			server.handleSignalQrCode(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for invalid device name '%s', got %d", tc.deviceName, w.Code)
			}

			if !strings.Contains(w.Body.String(), "invalid device name") {
				t.Errorf("Expected error message about invalid device name, got: %s", w.Body.String())
			}
		})
	}
}

func TestHandleSignalQrCode_SignalCLIError(t *testing.T) {
	// Setup mock Signal CLI server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Handle health check during server initialization
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Signal CLI error"))
	}))
	defer mockServer.Close()

	originalSignalURL := os.Getenv("SIGNAL_URL")
	defer func() {
		if originalSignalURL != "" {
			_ = os.Setenv("SIGNAL_URL", originalSignalURL)
		} else {
			_ = os.Unsetenv("SIGNAL_URL")
		}
	}()
	_ = os.Setenv("SIGNAL_URL", mockServer.URL[7:])

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	server := NewServer(":8080", testDB, nil)

	req := httptest.NewRequest("GET", "/api/signal/qrcode?device_name=test", nil)
	w := httptest.NewRecorder()

	server.handleSignalQrCode(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleSignalQrCode_NetworkTimeout(t *testing.T) {
	// Setup mock server that times out
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Handle health check during server initialization
			w.WriteHeader(http.StatusOK)
			return
		}
		time.Sleep(2 * time.Second) // Longer than our test timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	// Set a very short timeout for this test
	originalTimeout := os.Getenv("QR_TIMEOUT_SECONDS")
	defer func() {
		if originalTimeout != "" {
			_ = os.Setenv("QR_TIMEOUT_SECONDS", originalTimeout)
		} else {
			_ = os.Unsetenv("QR_TIMEOUT_SECONDS")
		}
	}()
	_ = os.Setenv("QR_TIMEOUT_SECONDS", "1")

	originalSignalURL := os.Getenv("SIGNAL_URL")
	defer func() {
		if originalSignalURL != "" {
			_ = os.Setenv("SIGNAL_URL", originalSignalURL)
		} else {
			_ = os.Unsetenv("SIGNAL_URL")
		}
	}()
	_ = os.Setenv("SIGNAL_URL", mockServer.URL[7:])

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	server := NewServer(":8080", testDB, nil)

	req := httptest.NewRequest("GET", "/api/signal/qrcode?device_name=test", nil)
	w := httptest.NewRecorder()

	server.handleSignalQrCode(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("Expected status 502 (timeout), got %d", w.Code)
	}
}

func TestHandleSignalQrCode_HeaderSecurity(t *testing.T) {
	// Setup mock Signal CLI server that returns various headers
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Handle health check during server initialization
			w.WriteHeader(http.StatusOK)
			return
		}
		// Safe headers (should be copied)
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("ETag", "test-etag")

		// Unsafe headers (should NOT be copied)
		w.Header().Set("Authorization", "Bearer secret-token")
		w.Header().Set("Set-Cookie", "session=secret")
		w.Header().Set("X-API-Key", "secret-key")
		w.Header().Set("Server", "signal-cli/1.0")

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("qr-code"))
	}))
	defer mockServer.Close()

	originalSignalURL := os.Getenv("SIGNAL_URL")
	defer func() {
		if originalSignalURL != "" {
			_ = os.Setenv("SIGNAL_URL", originalSignalURL)
		} else {
			_ = os.Unsetenv("SIGNAL_URL")
		}
	}()
	_ = os.Setenv("SIGNAL_URL", mockServer.URL[7:])

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	server := NewServer(":8080", testDB, nil)

	req := httptest.NewRequest("GET", "/api/signal/qrcode?device_name=test", nil)
	w := httptest.NewRecorder()

	server.handleSignalQrCode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that safe headers were copied
	if w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("Safe header Content-Type was not copied")
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Safe header Cache-Control was not copied")
	}
	if w.Header().Get("ETag") != "test-etag" {
		t.Errorf("Safe header ETag was not copied")
	}

	// Check that unsafe headers were NOT copied
	if w.Header().Get("Authorization") != "" {
		t.Errorf("Unsafe header Authorization was copied: %s", w.Header().Get("Authorization"))
	}
	if w.Header().Get("Set-Cookie") != "" {
		t.Errorf("Unsafe header Set-Cookie was copied: %s", w.Header().Get("Set-Cookie"))
	}
	if w.Header().Get("X-API-Key") != "" {
		t.Errorf("Unsafe header X-API-Key was copied: %s", w.Header().Get("X-API-Key"))
	}
	if w.Header().Get("Server") != "" {
		t.Errorf("Unsafe header Server was copied: %s", w.Header().Get("Server"))
	}
}

func TestHandleSignalQrCode_ResponseSizeLimit(t *testing.T) {
	// Setup mock server that returns a large response
	largeResponse := strings.Repeat("a", int(maxResponseSize)+1000) // Larger than limit
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/health" {
			// Handle health check during server initialization
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(largeResponse))
	}))
	defer mockServer.Close()

	originalSignalURL := os.Getenv("SIGNAL_URL")
	defer func() {
		if originalSignalURL != "" {
			_ = os.Setenv("SIGNAL_URL", originalSignalURL)
		} else {
			_ = os.Unsetenv("SIGNAL_URL")
		}
	}()
	_ = os.Setenv("SIGNAL_URL", mockServer.URL[7:])

	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	server := NewServer(":8080", testDB, nil)

	req := httptest.NewRequest("GET", "/api/signal/qrcode?device_name=test", nil)
	w := httptest.NewRecorder()

	server.handleSignalQrCode(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Response should be truncated to the limit
	responseSize := len(w.Body.Bytes())
	if responseSize > int(maxResponseSize) {
		t.Errorf("Response size %d exceeds limit %d", responseSize, maxResponseSize)
	}

	// Should be exactly the limit since we sent more data
	if responseSize != int(maxResponseSize) {
		t.Errorf("Expected response size to be exactly %d (truncated), got %d", maxResponseSize, responseSize)
	}
}

func TestHandleSignalQrCode_MethodNotAllowed(t *testing.T) {
	testDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer func() {
		_ = testDB.Close()
	}()

	// Disable Signal validation for this test to avoid network calls
	server := NewServerWithOptions(":8080", testDB, nil, WithSignalValidation(false))

	// Test POST method (should be rejected)
	req := httptest.NewRequest("POST", "/api/signal/qrcode", nil)
	w := httptest.NewRecorder()

	server.handleSignalQrCode(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for POST method, got %d", w.Code)
	}
}

func TestValidateDeviceName(t *testing.T) {
	testCases := []struct {
		name        string
		deviceName  string
		expectError bool
	}{
		{"valid alphanumeric", "device123", false},
		{"valid with hyphens", "my-device", false},
		{"valid with underscores", "my_device", false},
		{"valid mixed", "device-123_test", false},
		{"empty string", "", true},
		{"too long", strings.Repeat("a", 51), true},
		{"special characters", "device@name", true},
		{"spaces", "device name", true},
		{"sql injection", "device'; DROP TABLE users; --", true},
		{"url injection", "device&param=value", true},
		{"path traversal", "../device", true},
		{"dots", "device.name", true},
		{"slashes", "device/name", true},
		{"backslashes", "device\\name", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDeviceName(tc.deviceName)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for device name '%s', but got none", tc.deviceName)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for device name '%s', but got: %v", tc.deviceName, err)
			}
		})
	}
}

func TestGetQRTimeout(t *testing.T) {
	// Test default timeout
	originalTimeout := os.Getenv("QR_TIMEOUT_SECONDS")
	defer func() {
		if originalTimeout != "" {
			_ = os.Setenv("QR_TIMEOUT_SECONDS", originalTimeout)
		} else {
			_ = os.Unsetenv("QR_TIMEOUT_SECONDS")
		}
	}()

	_ = os.Unsetenv("QR_TIMEOUT_SECONDS")
	timeout := getQRTimeout()
	if timeout != defaultQRTimeout {
		t.Errorf("Expected default timeout %v, got %v", defaultQRTimeout, timeout)
	}

	// Test custom timeout
	_ = os.Setenv("QR_TIMEOUT_SECONDS", "60")
	timeout = getQRTimeout()
	expected := 60 * time.Second
	if timeout != expected {
		t.Errorf("Expected custom timeout %v, got %v", expected, timeout)
	}

	// Test invalid timeout (should fall back to default)
	_ = os.Setenv("QR_TIMEOUT_SECONDS", "invalid")
	timeout = getQRTimeout()
	if timeout != defaultQRTimeout {
		t.Errorf("Expected default timeout for invalid value, got %v", timeout)
	}

	// Test zero timeout (should fall back to default)
	_ = os.Setenv("QR_TIMEOUT_SECONDS", "0")
	timeout = getQRTimeout()
	if timeout != defaultQRTimeout {
		t.Errorf("Expected default timeout for zero value, got %v", timeout)
	}
}

func TestIsSafeHeader(t *testing.T) {
	safeHeaders := []string{
		"Content-Type",
		"content-type",
		"CONTENT-TYPE",
		"Content-Length",
		"Cache-Control",
		"ETag",
		"Last-Modified",
		"Expires",
		"Content-Encoding",
	}

	for _, header := range safeHeaders {
		if !isSafeHeader(header) {
			t.Errorf("Header '%s' should be safe but was marked unsafe", header)
		}
	}

	unsafeHeaders := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"X-Forwarded-For",
		"X-Real-IP",
		"X-API-Key",
		"Server",
		"authorization",
		"COOKIE",
		"set-cookie",
	}

	for _, header := range unsafeHeaders {
		if isSafeHeader(header) {
			t.Errorf("Header '%s' should be unsafe but was marked safe", header)
		}
	}

	// Test unknown headers (should be unsafe)
	unknownHeaders := []string{
		"X-Custom-Header",
		"Unknown-Header",
		"Random-Header",
	}

	for _, header := range unknownHeaders {
		if isSafeHeader(header) {
			t.Errorf("Unknown header '%s' should be unsafe but was marked safe", header)
		}
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
