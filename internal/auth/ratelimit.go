package auth

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	clients map[string]*Client
	mutex   sync.RWMutex
	
	// Configuration
	maxRequests int           // Maximum requests per window
	window      time.Duration // Time window
	cleanup     time.Duration // Cleanup interval for expired clients
}

// Client represents a rate-limited client
type Client struct {
	requests  int       // Current request count
	window    time.Time // Current window start time
	lastSeen  time.Time // Last request time (for cleanup)
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxRequests int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:     make(map[string]*Client),
		maxRequests: maxRequests,
		window:      window,
		cleanup:     window * 2, // Cleanup expired clients every 2 windows
	}
	
	// Start cleanup goroutine
	go rl.cleanupExpiredClients()
	
	return rl
}

// IsAllowed checks if a request from the given client ID is allowed
func (rl *RateLimiter) IsAllowed(clientID string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	client, exists := rl.clients[clientID]
	if !exists {
		// New client
		rl.clients[clientID] = &Client{
			requests: 1,
			window:   now,
			lastSeen: now,
		}
		return true
	}
	
	client.lastSeen = now
	
	// Check if we're in a new window
	if now.Sub(client.window) >= rl.window {
		// Reset for new window
		client.requests = 1
		client.window = now
		return true
	}
	
	// Check if we've exceeded the limit
	if client.requests >= rl.maxRequests {
		return false
	}
	
	// Increment and allow
	client.requests++
	return true
}

// cleanupExpiredClients removes clients that haven't made requests recently
func (rl *RateLimiter) cleanupExpiredClients() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mutex.Lock()
		
		now := time.Now()
		for clientID, client := range rl.clients {
			if now.Sub(client.lastSeen) > rl.cleanup {
				delete(rl.clients, clientID)
			}
		}
		
		rl.mutex.Unlock()
	}
}

// AuthRateLimiter provides rate limiting for authentication endpoints
type AuthRateLimiter struct {
	limiter *RateLimiter
}

// NewAuthRateLimiter creates a new authentication rate limiter
// Default: 10 requests per minute per IP
func NewAuthRateLimiter() *AuthRateLimiter {
	return &AuthRateLimiter{
		limiter: NewRateLimiter(10, time.Minute),
	}
}

// NewCustomAuthRateLimiter creates a new authentication rate limiter with custom limits
func NewCustomAuthRateLimiter(maxRequests int, window time.Duration) *AuthRateLimiter {
	return &AuthRateLimiter{
		limiter: NewRateLimiter(maxRequests, window),
	}
}

// getClientID extracts a client identifier from the request
func (arl *AuthRateLimiter) getClientID(r *http.Request) string {
	// Use X-Forwarded-For header if available (for reverse proxy setups)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	
	// Use X-Real-IP header if available
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// Middleware returns HTTP middleware that enforces rate limiting
func (arl *AuthRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID := arl.getClientID(r)
		
		if !arl.limiter.IsAllowed(clientID) {
			// Log rate limit violation
			slog.WarnContext(r.Context(), "Rate limit exceeded",
				slog.String("event", "rate_limit_violation"),
				slog.String("client_ip", clientID),
				slog.String("user_agent", r.Header.Get("User-Agent")),
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.Time("timestamp", time.Now()),
			)
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"Too many requests. Please try again later."}}`))
			return
		}
		
		next.ServeHTTP(w, r)
	})
}