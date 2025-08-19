package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const (
	CSRFTokenKey    = "csrf_token"
	CSRFHeaderName  = "X-CSRF-Token"
	CSRFTokenLength = 32
)

// CSRFProtection provides CSRF protection middleware
type CSRFProtection struct {
	sessionManager *SessionManager
}

// NewCSRFProtection creates a new CSRF protection middleware
func NewCSRFProtection(sessionManager *SessionManager) *CSRFProtection {
	return &CSRFProtection{
		sessionManager: sessionManager,
	}
}

// generateCSRFToken generates a random CSRF token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GetCSRFToken retrieves or generates a CSRF token for the current session
func (csrf *CSRFProtection) GetCSRFToken(r *http.Request) (string, error) {
	// Try to get existing token from session
	token := csrf.sessionManager.Manager.GetString(r.Context(), CSRFTokenKey)
	if token != "" {
		return token, nil
	}
	
	// Generate new token if none exists
	newToken, err := generateCSRFToken()
	if err != nil {
		return "", err
	}
	
	// Store in session
	csrf.sessionManager.Manager.Put(r.Context(), CSRFTokenKey, newToken)
	return newToken, nil
}

// ValidateCSRFToken validates the CSRF token from request headers
func (csrf *CSRFProtection) ValidateCSRFToken(r *http.Request) bool {
	// Get token from session
	sessionToken := csrf.sessionManager.Manager.GetString(r.Context(), CSRFTokenKey)
	if sessionToken == "" {
		return false
	}
	
	// Get token from header
	headerToken := r.Header.Get(CSRFHeaderName)
	if headerToken == "" {
		return false
	}
	
	// Compare tokens (constant time comparison to prevent timing attacks)
	return subtle.ConstantTimeCompare([]byte(sessionToken), []byte(headerToken)) == 1
}

// Middleware returns HTTP middleware that validates CSRF tokens for state-changing operations
func (csrf *CSRFProtection) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only check CSRF for state-changing methods
		if isStateChangingMethod(r.Method) {
			if !csrf.ValidateCSRFToken(r) {
				// Log CSRF violation
				clientIP := r.Header.Get("X-Forwarded-For")
				if clientIP == "" {
					clientIP = r.Header.Get("X-Real-IP")
				}
				if clientIP == "" {
					clientIP = r.RemoteAddr
				}
				
				slog.WarnContext(r.Context(), "CSRF token validation failed",
					slog.String("event", "csrf_violation"),
					slog.String("client_ip", clientIP),
					slog.String("user_agent", r.Header.Get("User-Agent")),
					slog.String("path", r.URL.Path),
					slog.String("method", r.Method),
					slog.Time("timestamp", time.Now()),
				)
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":{"code":"CSRF_TOKEN_INVALID","message":"CSRF token validation failed"}}`))
				return
			}
		}
		
		next.ServeHTTP(w, r)
	})
}

// isStateChangingMethod returns true if the HTTP method can change state
func isStateChangingMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

