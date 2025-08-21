package auth

import (
	"net/http"
)

func (sm *SessionManager) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := sm.Manager.GetInt(r.Context(), "user_id")
		if userID == 0 {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (sm *SessionManager) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Just load session, don't require auth
		next.ServeHTTP(w, r)
	})
}