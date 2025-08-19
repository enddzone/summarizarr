package auth

import (
	"log/slog"
	"net/http"
	"time"
)

// AuthLogger provides structured logging for authentication events
type AuthLogger struct {
	logger *slog.Logger
}

// NewAuthLogger creates a new authentication logger
func NewAuthLogger() *AuthLogger {
	return &AuthLogger{
		logger: slog.Default(),
	}
}

// getClientInfo extracts client information from the request for logging
func (al *AuthLogger) getClientInfo(r *http.Request) (string, string) {
	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}
	
	// Get user agent
	userAgent := r.Header.Get("User-Agent")
	
	return clientIP, userAgent
}

// LogLoginAttempt logs a login attempt
func (al *AuthLogger) LogLoginAttempt(r *http.Request, email string, success bool, reason string) {
	clientIP, userAgent := al.getClientInfo(r)
	
	logArgs := []any{
		slog.String("event", "login_attempt"),
		slog.String("email", email),
		slog.Bool("success", success),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.Time("timestamp", time.Now()),
	}
	
	if reason != "" {
		logArgs = append(logArgs, slog.String("reason", reason))
	}
	
	if success {
		al.logger.InfoContext(r.Context(), "User login successful", logArgs...)
	} else {
		al.logger.WarnContext(r.Context(), "User login failed", logArgs...)
	}
}

// LogRegistration logs a user registration attempt
func (al *AuthLogger) LogRegistration(r *http.Request, email string, success bool, reason string) {
	clientIP, userAgent := al.getClientInfo(r)
	
	logArgs := []any{
		slog.String("event", "registration_attempt"),
		slog.String("email", email),
		slog.Bool("success", success),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.Time("timestamp", time.Now()),
	}
	
	if reason != "" {
		logArgs = append(logArgs, slog.String("reason", reason))
	}
	
	if success {
		al.logger.InfoContext(r.Context(), "User registration successful", logArgs...)
	} else {
		al.logger.WarnContext(r.Context(), "User registration failed", logArgs...)
	}
}

// LogLogout logs a user logout
func (al *AuthLogger) LogLogout(r *http.Request, userEmail string, success bool) {
	clientIP, userAgent := al.getClientInfo(r)
	
	logArgs := []any{
		slog.String("event", "logout"),
		slog.String("email", userEmail),
		slog.Bool("success", success),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.Time("timestamp", time.Now()),
	}
	
	if success {
		al.logger.InfoContext(r.Context(), "User logout successful", logArgs...)
	} else {
		al.logger.ErrorContext(r.Context(), "User logout failed", logArgs...)
	}
}

// LogSessionCreation logs session creation
func (al *AuthLogger) LogSessionCreation(r *http.Request, userEmail string, userID int) {
	clientIP, userAgent := al.getClientInfo(r)
	
	al.logger.InfoContext(r.Context(), "Session created",
		slog.String("event", "session_created"),
		slog.String("email", userEmail),
		slog.Int("user_id", userID),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.Time("timestamp", time.Now()),
	)
}

// LogSessionDestroyed logs session destruction
func (al *AuthLogger) LogSessionDestroyed(r *http.Request, userEmail string) {
	clientIP, userAgent := al.getClientInfo(r)
	
	al.logger.InfoContext(r.Context(), "Session destroyed",
		slog.String("event", "session_destroyed"),
		slog.String("email", userEmail),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.Time("timestamp", time.Now()),
	)
}

// LogAuthMiddlewareAction logs authentication middleware actions
func (al *AuthLogger) LogAuthMiddlewareAction(r *http.Request, action string, success bool, userEmail string) {
	clientIP, userAgent := al.getClientInfo(r)
	
	logArgs := []any{
		slog.String("event", "auth_middleware"),
		slog.String("action", action),
		slog.Bool("success", success),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Time("timestamp", time.Now()),
	}
	
	if userEmail != "" {
		logArgs = append(logArgs, slog.String("email", userEmail))
	}
	
	if success {
		al.logger.DebugContext(r.Context(), "Authentication middleware action", logArgs...)
	} else {
		al.logger.WarnContext(r.Context(), "Authentication middleware blocked request", logArgs...)
	}
}

// LogCSRFViolation logs CSRF token violations
func (al *AuthLogger) LogCSRFViolation(r *http.Request) {
	clientIP, userAgent := al.getClientInfo(r)
	
	al.logger.WarnContext(r.Context(), "CSRF token validation failed",
		slog.String("event", "csrf_violation"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Time("timestamp", time.Now()),
	)
}

// LogRateLimitViolation logs rate limit violations
func (al *AuthLogger) LogRateLimitViolation(r *http.Request) {
	clientIP, userAgent := al.getClientInfo(r)
	
	al.logger.WarnContext(r.Context(), "Rate limit exceeded",
		slog.String("event", "rate_limit_violation"),
		slog.String("client_ip", clientIP),
		slog.String("user_agent", userAgent),
		slog.String("path", r.URL.Path),
		slog.String("method", r.Method),
		slog.Time("timestamp", time.Now()),
	)
}