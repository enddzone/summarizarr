package api

import (
	"encoding/json"
	"net/http"
	"summarizarr/internal/auth"
)

type AuthHandlers struct {
	userStore      *auth.UserStore
	sessionManager *auth.SessionManager
	csrfProtection *auth.CSRFProtection
	logger         *auth.AuthLogger
}

func NewAuthHandlers(userStore *auth.UserStore, sessionManager *auth.SessionManager) *AuthHandlers {
	return &AuthHandlers{
		userStore:      userStore,
		sessionManager: sessionManager,
		csrfProtection: auth.NewCSRFProtection(sessionManager),
		logger:         auth.NewAuthLogger(),
	}
}

// POST /api/auth/login
func (ah *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowedError(w, "POST")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvalidInputError(w, "Invalid JSON format")
		return
	}

	// Validate and sanitize input
	email, password, validation := auth.ValidateAndSanitizeLoginRequest(req.Email, req.Password)
	if !validation.Valid {
		// Convert validation errors to the expected format
		var fieldErrors []map[string]interface{}
		for _, err := range validation.Errors {
			fieldErrors = append(fieldErrors, map[string]interface{}{
				"field":   err.Field,
				"message": err.Message,
			})
		}
		writeValidationErrorResponse(w, fieldErrors)
		return
	}

	user, err := ah.userStore.ValidateUser(email, password)
	if err != nil {
		ah.logger.LogLoginAttempt(r, email, false, "invalid_credentials")
		writeInvalidCredentialsError(w)
		return
	}

	// Create session
	ah.sessionManager.Manager.Put(r.Context(), "user_id", user.ID)
	ah.sessionManager.Manager.Put(r.Context(), "user_email", user.Email)
	
	// Log successful login and session creation
	ah.logger.LogLoginAttempt(r, email, true, "")
	ah.logger.LogSessionCreation(r, user.Email, user.ID)

	// Return user info (no sensitive data)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		},
		"message": "Login successful",
	}); err != nil {
		writeInternalServerError(w, "Failed to encode response")
		return
	}
}

// POST /api/auth/logout
func (ah *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowedError(w, "POST")
		return
	}

	// Get user email for logging before destroying session
	userEmail := ah.sessionManager.Manager.GetString(r.Context(), "user_email")

	err := ah.sessionManager.Manager.Destroy(r.Context())
	if err != nil {
		ah.logger.LogLogout(r, userEmail, false)
		writeInternalServerError(w, "Failed to destroy session")
		return
	}
	
	// Log successful logout
	ah.logger.LogLogout(r, userEmail, true)
	if userEmail != "" {
		ah.logger.LogSessionDestroyed(r, userEmail)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"message": "Logout successful"}); err != nil {
		writeInternalServerError(w, "Failed to encode response")
		return
	}
}

// GET /api/auth/me
func (ah *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowedError(w, "GET")
		return
	}

	userID := ah.sessionManager.Manager.GetInt(r.Context(), "user_id")
	if userID == 0 {
		writeAuthRequiredError(w)
		return
	}

	user, err := ah.userStore.GetUser(userID)
	if err != nil {
		writeUserNotFoundError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		},
	}); err != nil {
		writeInternalServerError(w, "Failed to encode response")
		return
	}
}

// POST /api/auth/register
func (ah *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowedError(w, "POST")
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInvalidInputError(w, "Invalid JSON format")
		return
	}

	// Validate and sanitize input
	email, password, validation := auth.ValidateAndSanitizeRegisterRequest(req.Email, req.Password)
	if !validation.Valid {
		// Convert validation errors to the expected format
		var fieldErrors []map[string]interface{}
		for _, err := range validation.Errors {
			fieldErrors = append(fieldErrors, map[string]interface{}{
				"field":   err.Field,
				"message": err.Message,
			})
		}
		ah.logger.LogRegistration(r, req.Email, false, "validation_failed")
		writeValidationErrorResponse(w, fieldErrors)
		return
	}

	user, err := ah.userStore.CreateUser(email, password)
	if err != nil {
		ah.logger.LogRegistration(r, email, false, "user_exists")
		writeUserExistsError(w)
		return
	}

	// Create session for new user
	ah.sessionManager.Manager.Put(r.Context(), "user_id", user.ID)
	ah.sessionManager.Manager.Put(r.Context(), "user_email", user.Email)
	
	// Log successful registration and session creation
	ah.logger.LogRegistration(r, email, true, "")
	ah.logger.LogSessionCreation(r, user.Email, user.ID)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		},
		"message": "Registration successful",
	}); err != nil {
		writeInternalServerError(w, "Failed to encode response")
		return
	}
}

// GET /api/auth/csrf-token
func (ah *AuthHandlers) CSRFToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowedError(w, "GET")
		return
	}

	token, err := ah.csrfProtection.GetCSRFToken(r)
	if err != nil {
		writeInternalServerError(w, "Failed to generate CSRF token")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"csrf_token": token}); err != nil {
		writeInternalServerError(w, "Failed to encode response")
		return
	}
}
