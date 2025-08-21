package auth

import (
	"database/sql"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
)

type SessionManager struct {
	Manager *scs.SessionManager
}

func NewSessionManager(db *sql.DB) *SessionManager {
	sessionManager := scs.New()
	sessionManager.Store = sqlite3store.New(db)
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Cookie.Name = "summarizarr_session"
	sessionManager.Cookie.HttpOnly = true
	
	// Environment-based cookie security
	sessionManager.Cookie.Secure = isProduction()
	sessionManager.Cookie.SameSite = http.SameSiteLaxMode
	sessionManager.Cookie.Path = "/"

	return &SessionManager{Manager: sessionManager}
}

// isProduction determines if we're running in production based on environment variables
func isProduction() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if env == "production" || env == "prod" {
		return true
	}
	
	// Check for explicit COOKIE_SECURE setting
	if cookieSecure := strings.ToLower(os.Getenv("COOKIE_SECURE")); cookieSecure != "" {
		return cookieSecure == "true" || cookieSecure == "1"
	}
	
	// Default to false for development
	return false
}