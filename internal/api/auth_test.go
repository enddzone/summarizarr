package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"summarizarr/internal/auth"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupAuthTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
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

func setupAuthHandlers(t *testing.T) (*AuthHandlers, *auth.SessionManager) {
	db := setupAuthTestDB(t)
	sessionManager := auth.NewSessionManager(db)
	userStore := auth.NewUserStore(db)
	return NewAuthHandlers(userStore, sessionManager), sessionManager
}

func TestAuthHandlers(t *testing.T) {
	authHandlers, sessionManager := setupAuthHandlers(t)

	t.Run("Register", func(t *testing.T) {
		t.Run("ValidRegistration", func(t *testing.T) {
			reqBody := map[string]string{
				"email":    "test@example.com",
				"password": "SecurePass123!",
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Load session context
			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Register(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response["message"] != "Registration successful" {
				t.Errorf("Expected success message, got %v", response["message"])
			}

			user, exists := response["user"].(map[string]interface{})
			if !exists {
				t.Error("Expected user object in response")
			} else {
				if user["email"] != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got %v", user["email"])
				}
			}
		})

		t.Run("InvalidPassword", func(t *testing.T) {
			reqBody := map[string]string{
				"email":    "weak@example.com",
				"password": "weak",
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Register(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if errorObj, ok := response["error"].(map[string]interface{}); ok {
				if errorObj["code"] != "VALIDATION_FAILED" {
					t.Errorf("Expected VALIDATION_FAILED error code, got %v", errorObj["code"])
				}
			} else {
				t.Errorf("Expected error object in response, got %v", response["error"])
			}
		})

		t.Run("InvalidEmail", func(t *testing.T) {
			reqBody := map[string]string{
				"email":    "invalid-email",
				"password": "SecurePass123!",
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Register(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})

		t.Run("MethodNotAllowed", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/auth/register", nil)
			w := httptest.NewRecorder()

			authHandlers.Register(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}
		})
	})

	t.Run("Login", func(t *testing.T) {
		// First register a user
		userStore := auth.NewUserStore(setupAuthTestDB(t))
		userStore.CreateUser("login@example.com", "SecurePass123!")

		t.Run("ValidLogin", func(t *testing.T) {
			// Create new auth handlers and set up a user
			authHandlers, sessionManager := setupAuthHandlers(t)
			authHandlers.userStore.CreateUser("login@example.com", "SecurePass123!")

			reqBody := map[string]string{
				"email":    "login@example.com",
				"password": "SecurePass123!",
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Login(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response["message"] != "Login successful" {
				t.Errorf("Expected success message, got %v", response["message"])
			}
		})

		t.Run("InvalidCredentials", func(t *testing.T) {
			// Create new auth handlers and set up a user
			authHandlers, sessionManager := setupAuthHandlers(t)
			authHandlers.userStore.CreateUser("login@example.com", "SecurePass123!")

			reqBody := map[string]string{
				"email":    "login@example.com",
				"password": "wrongpassword",
			}
			jsonBody, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Login(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", w.Code)
			}
		})

		t.Run("InvalidJSON", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader("invalid json"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			authHandlers.Login(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		})
	})

	t.Run("CSRFToken", func(t *testing.T) {
		t.Run("GetCSRFToken", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/auth/csrf-token", nil)
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.CSRFToken(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			token, exists := response["csrf_token"]
			if !exists || token == "" {
				t.Error("Expected non-empty CSRF token in response")
			}
		})

		t.Run("MethodNotAllowed", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/csrf-token", nil)
			w := httptest.NewRecorder()

			authHandlers.CSRFToken(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}
		})
	})

	t.Run("Logout", func(t *testing.T) {
		t.Run("ValidLogout", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/logout", nil)
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Logout(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			if response["message"] != "Logout successful" {
				t.Errorf("Expected success message, got %v", response["message"])
			}
		})

		t.Run("MethodNotAllowed", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/auth/logout", nil)
			w := httptest.NewRecorder()

			authHandlers.Logout(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}
		})
	})

	t.Run("Me", func(t *testing.T) {
		t.Run("AuthenticatedUser", func(t *testing.T) {
			// First register and login a user
			authHandlers, sessionManager := setupAuthHandlers(t)
			user, _ := authHandlers.userStore.CreateUser("me@example.com", "SecurePass123!")

			req := httptest.NewRequest("GET", "/api/auth/me", nil)
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			
			// Set user session
			sessionManager.Manager.Put(ctx, "user_id", user.ID)
			req = req.WithContext(ctx)

			authHandlers.Me(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			userInfo, exists := response["user"].(map[string]interface{})
			if !exists {
				t.Error("Expected user object in response")
			} else {
				if userInfo["email"] != "me@example.com" {
					t.Errorf("Expected email 'me@example.com', got %v", userInfo["email"])
				}
			}
		})

		t.Run("UnauthenticatedUser", func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/auth/me", nil)
			w := httptest.NewRecorder()

			ctx, err := sessionManager.Manager.Load(req.Context(), "")
			if err != nil {
				t.Fatalf("Failed to load session: %v", err)
			}
			req = req.WithContext(ctx)

			authHandlers.Me(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", w.Code)
			}
		})

		t.Run("MethodNotAllowed", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/me", nil)
			w := httptest.NewRecorder()

			authHandlers.Me(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status 405, got %d", w.Code)
			}
		})
	})
}