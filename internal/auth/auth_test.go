package auth

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create auth_users table
	_, err = db.Exec(`
		CREATE TABLE auth_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create auth_users table: %v", err)
	}

	// Create sessions table
	_, err = db.Exec(`
		CREATE TABLE sessions (
			token TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			expiry REAL NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}

	return db
}

func TestUserStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userStore := NewUserStore(db)

	t.Run("CreateUser", func(t *testing.T) {
		user, err := userStore.CreateUser("test@example.com", "SecurePass123!")
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		if user.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
		}

		if user.ID == 0 {
			t.Error("Expected user ID to be set")
		}
	})

	t.Run("CreateUserDuplicateEmail", func(t *testing.T) {
		_, err := userStore.CreateUser("test@example.com", "AnotherPass123!")
		if err == nil {
			t.Error("Expected error when creating user with duplicate email")
		}
	})

	t.Run("ValidateUser", func(t *testing.T) {
		user, err := userStore.ValidateUser("test@example.com", "SecurePass123!")
		if err != nil {
			t.Fatalf("Failed to validate user: %v", err)
		}

		if user.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got '%s'", user.Email)
		}
	})

	t.Run("ValidateUserInvalidPassword", func(t *testing.T) {
		_, err := userStore.ValidateUser("test@example.com", "wrongpassword")
		if err == nil {
			t.Error("Expected error for invalid password")
		}
	})

	t.Run("ValidateUserNonexistentEmail", func(t *testing.T) {
		_, err := userStore.ValidateUser("nonexistent@example.com", "password")
		if err == nil {
			t.Error("Expected error for nonexistent email")
		}
	})

	t.Run("GetUser", func(t *testing.T) {
		// First create and get the user ID
		user, err := userStore.CreateUser("gettest@example.com", "SecurePass123!")
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		retrievedUser, err := userStore.GetUser(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrievedUser.Email != "gettest@example.com" {
			t.Errorf("Expected email 'gettest@example.com', got '%s'", retrievedUser.Email)
		}
	})
}

func TestSessionManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionManager := NewSessionManager(db)

	t.Run("CookieConfiguration", func(t *testing.T) {
		// Test cookie security in development (should be false)
		if sessionManager.Manager.Cookie.Secure {
			t.Error("Expected cookie.Secure to be false in development")
		}

		if !sessionManager.Manager.Cookie.HttpOnly {
			t.Error("Expected cookie.HttpOnly to be true")
		}

		if sessionManager.Manager.Cookie.SameSite != http.SameSiteLaxMode {
			t.Error("Expected cookie.SameSite to be Lax")
		}
	})

	t.Run("SessionOperations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)

		// Load session
		ctx, err := sessionManager.Manager.Load(req.Context(), "")
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}

		// Put value in session
		sessionManager.Manager.Put(ctx, "test_key", "test_value")

		// Get value from session
		value := sessionManager.Manager.GetString(ctx, "test_key")
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}
	})
}

func TestCSRFProtection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	sessionManager := NewSessionManager(db)
	csrfProtection := NewCSRFProtection(sessionManager)

	t.Run("GenerateCSRFToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		
		// Load session context
		ctx, err := sessionManager.Manager.Load(req.Context(), "")
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}
		req = req.WithContext(ctx)

		token, err := csrfProtection.GetCSRFToken(req)
		if err != nil {
			t.Fatalf("Failed to generate CSRF token: %v", err)
		}

		if token == "" {
			t.Error("Expected non-empty CSRF token")
		}

		// Second call should return the same token
		token2, err := csrfProtection.GetCSRFToken(req)
		if err != nil {
			t.Fatalf("Failed to get CSRF token: %v", err)
		}

		if token != token2 {
			t.Error("Expected same CSRF token on subsequent calls")
		}
	})

	t.Run("ValidateCSRFToken", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", nil)
		
		// Load session context
		ctx, err := sessionManager.Manager.Load(req.Context(), "")
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}
		req = req.WithContext(ctx)

		// Generate token
		token, err := csrfProtection.GetCSRFToken(req)
		if err != nil {
			t.Fatalf("Failed to generate CSRF token: %v", err)
		}

		// Set token in header and validate
		req.Header.Set(CSRFHeaderName, token)
		if !csrfProtection.ValidateCSRFToken(req) {
			t.Error("Expected CSRF token validation to pass")
		}

		// Test with wrong token
		req.Header.Set(CSRFHeaderName, "wrong-token")
		if csrfProtection.ValidateCSRFToken(req) {
			t.Error("Expected CSRF token validation to fail with wrong token")
		}
	})

	t.Run("CSRFMiddleware", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := csrfProtection.Middleware(handler)

		// Test GET request (should pass without CSRF token)
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		
		ctx, err := sessionManager.Manager.Load(req.Context(), "")
		if err != nil {
			t.Fatalf("Failed to load session: %v", err)
		}
		req = req.WithContext(ctx)

		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for GET request, got %d", w.Code)
		}

		// Test POST request without CSRF token (should fail)
		req = httptest.NewRequest("POST", "/", nil)
		req = req.WithContext(ctx)
		w = httptest.NewRecorder()

		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for POST without CSRF token, got %d", w.Code)
		}
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("BasicRateLimiting", func(t *testing.T) {
		limiter := NewRateLimiter(2, time.Minute)

		// First two requests should be allowed
		if !limiter.IsAllowed("client1") {
			t.Error("Expected first request to be allowed")
		}

		if !limiter.IsAllowed("client1") {
			t.Error("Expected second request to be allowed")
		}

		// Third request should be denied
		if limiter.IsAllowed("client1") {
			t.Error("Expected third request to be denied")
		}
	})

	t.Run("DifferentClients", func(t *testing.T) {
		limiter := NewRateLimiter(1, time.Minute)

		// Each client should have their own limit
		if !limiter.IsAllowed("client1") {
			t.Error("Expected request from client1 to be allowed")
		}

		if !limiter.IsAllowed("client2") {
			t.Error("Expected request from client2 to be allowed")
		}

		// Both clients should now be at their limit
		if limiter.IsAllowed("client1") {
			t.Error("Expected second request from client1 to be denied")
		}

		if limiter.IsAllowed("client2") {
			t.Error("Expected second request from client2 to be denied")
		}
	})

	t.Run("WindowReset", func(t *testing.T) {
		limiter := NewRateLimiter(1, time.Millisecond*100)

		// Use up the limit
		if !limiter.IsAllowed("client1") {
			t.Error("Expected first request to be allowed")
		}

		if limiter.IsAllowed("client1") {
			t.Error("Expected second request to be denied")
		}

		// Wait for window to reset
		time.Sleep(time.Millisecond * 150)

		// Should be allowed again
		if !limiter.IsAllowed("client1") {
			t.Error("Expected request after window reset to be allowed")
		}
	})
}

func TestAuthRateLimiter(t *testing.T) {
	rateLimiter := NewAuthRateLimiter()

	t.Run("RateLimitMiddleware", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := rateLimiter.Middleware(handler)

		// Create requests from the same "client" - should allow 10 requests
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("POST", "/login", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for request %d, got %d", i+1, w.Code)
			}
		}

		// 11th request should be rate limited
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429 for 11th request, got %d", w.Code)
		}
	})
}

func TestValidation(t *testing.T) {
	t.Run("ValidateEmail", func(t *testing.T) {
		tests := []struct {
			email string
			valid bool
		}{
			{"test@example.com", true},
			{"user+tag@domain.co.uk", true},
			{"invalid-email", false},
			{"", false},
			{"@domain.com", false},
			{"user@", false},
			{strings.Repeat("a", 250) + "@example.com", false}, // Too long
		}

		for _, test := range tests {
			result := ValidateEmail(test.email)
			if result.Valid != test.valid {
				t.Errorf("Expected validation of '%s' to be %v, got %v", test.email, test.valid, result.Valid)
			}
		}
	})

	t.Run("ValidatePassword", func(t *testing.T) {
		tests := []struct {
			password string
			valid    bool
		}{
			{"StrongPass123!", true},
			{"weak", false},                    // Too short
			{"NoNumber!", false},               // No number
			{"nonumber123", false},             // No uppercase
			{"NOLOWERCASE123!", false},         // No lowercase
			{"NoSpecialChar123", false},        // No special character
			{"", false},                        // Empty
			{strings.Repeat("a", 130), false}, // Too long
		}

		for _, test := range tests {
			result := ValidatePassword(test.password)
			if result.Valid != test.valid {
				t.Errorf("Expected validation of password to be %v, got %v. Errors: %v", test.valid, result.Valid, result.Errors)
			}
		}
	})

	t.Run("SanitizeInput", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"  test@example.com  ", "test@example.com"},
			{"test\x00email", "testemail"},
			{"test\r\nemail", "testemail"},
			{"normal-email@domain.com", "normal-email@domain.com"},
		}

		for _, test := range tests {
			result := SanitizeInput(test.input)
			if result != test.expected {
				t.Errorf("Expected sanitized input '%s', got '%s'", test.expected, result)
			}
		}
	})
}