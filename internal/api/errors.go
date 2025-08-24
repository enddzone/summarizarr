package api

import (
	"encoding/json"
	"net/http"
)

// APIError represents a structured API error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ErrorResponse represents the complete error response structure
type ErrorResponse struct {
	Error  APIError                 `json:"error"`
	Errors []map[string]interface{} `json:"errors,omitempty"` // For validation errors
}

// Error codes for different types of failures
const (
	// Authentication errors
	ErrCodeAuthRequired     = "AUTH_REQUIRED"
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeUserExists       = "USER_EXISTS"
	ErrCodeUserNotFound     = "USER_NOT_FOUND"
	ErrCodeSessionExpired   = "SESSION_EXPIRED"
	ErrCodeCSRFTokenInvalid = "CSRF_TOKEN_INVALID"
	
	// Validation errors
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeInvalidInput     = "INVALID_INPUT"
	ErrCodeMissingField     = "MISSING_FIELD"
	
	// Rate limiting
	ErrCodeRateLimited = "RATE_LIMITED"
	
	// Server errors
	ErrCodeInternalError   = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	
	// Resource errors
	ErrCodeNotFound      = "NOT_FOUND"
	ErrCodeAlreadyExists = "ALREADY_EXISTS"
	ErrCodeInvalidFormat = "INVALID_FORMAT"
)

// writeErrorResponse writes a structured error response
func writeErrorResponse(w http.ResponseWriter, status int, code, message string, details ...string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	apiError := APIError{
		Code:    code,
		Message: message,
	}
	
	if len(details) > 0 {
		apiError.Details = details[0]
	}
	
	response := ErrorResponse{
		Error: apiError,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error but don't try to write another response
		// since headers are already sent
		return
	}
}

// writeValidationErrorResponse writes a validation error response with field-specific errors
func writeValidationErrorResponse(w http.ResponseWriter, fieldErrors []map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	
	response := ErrorResponse{
		Error: APIError{
			Code:    ErrCodeValidationFailed,
			Message: "Input validation failed",
		},
		Errors: fieldErrors,
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Log the error but don't try to write another response
		// since headers are already sent
		return
	}
}

// Common error response helpers
func writeAuthRequiredError(w http.ResponseWriter) {
	writeErrorResponse(w, http.StatusUnauthorized, ErrCodeAuthRequired, "Authentication required")
}

func writeInvalidCredentialsError(w http.ResponseWriter) {
	writeErrorResponse(w, http.StatusUnauthorized, ErrCodeInvalidCredentials, "Invalid email or password")
}

func writeUserExistsError(w http.ResponseWriter) {
	writeErrorResponse(w, http.StatusConflict, ErrCodeUserExists, "User with this email already exists")
}

func writeUserNotFoundError(w http.ResponseWriter) {
	writeErrorResponse(w, http.StatusNotFound, ErrCodeUserNotFound, "User not found")
}

func writeMethodNotAllowedError(w http.ResponseWriter, allowedMethods ...string) {
	if len(allowedMethods) > 0 {
		w.Header().Set("Allow", allowedMethods[0])
	}
	writeErrorResponse(w, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "Method not allowed")
}

func writeInvalidInputError(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusBadRequest, ErrCodeInvalidInput, "Invalid request format", details)
}

func writeInternalServerError(w http.ResponseWriter, details string) {
	writeErrorResponse(w, http.StatusInternalServerError, ErrCodeInternalError, "Internal server error", details)
}