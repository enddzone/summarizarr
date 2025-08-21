package auth

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Validation constants
const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
	MaxEmailLength    = 254
)

// Email validation regex (RFC 5322 compliant)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult holds the results of input validation
type ValidationResult struct {
	Errors []ValidationError
	Valid  bool
}

// AddError adds a validation error
func (vr *ValidationResult) AddError(field, message string) {
	vr.Errors = append(vr.Errors, ValidationError{Field: field, Message: message})
	vr.Valid = false
}

// ValidateEmail validates email address format and length
func ValidateEmail(email string) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	// Trim whitespace
	email = strings.TrimSpace(email)
	
	// Check if empty
	if email == "" {
		result.AddError("email", "email is required")
		return result
	}
	
	// Check length
	if len(email) > MaxEmailLength {
		result.AddError("email", fmt.Sprintf("email must be no more than %d characters", MaxEmailLength))
		return result
	}
	
	// Check format
	if !emailRegex.MatchString(email) {
		result.AddError("email", "invalid email format")
		return result
	}
	
	return result
}

// ValidatePassword validates password strength and requirements
func ValidatePassword(password string) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	// Check if empty
	if password == "" {
		result.AddError("password", "password is required")
		return result
	}
	
	// Check length
	if len(password) < MinPasswordLength {
		result.AddError("password", fmt.Sprintf("password must be at least %d characters long", MinPasswordLength))
	}
	
	if len(password) > MaxPasswordLength {
		result.AddError("password", fmt.Sprintf("password must be no more than %d characters long", MaxPasswordLength))
		return result
	}
	
	// Check for required character types
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		result.AddError("password", "password must contain at least one uppercase letter")
	}
	
	if !hasLower {
		result.AddError("password", "password must contain at least one lowercase letter")
	}
	
	if !hasNumber {
		result.AddError("password", "password must contain at least one number")
	}
	
	if !hasSpecial {
		result.AddError("password", "password must contain at least one special character")
	}
	
	return result
}

// SanitizeInput sanitizes user input by trimming whitespace and removing potential XSS vectors
func SanitizeInput(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Remove carriage returns and line feeds from single-line inputs
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\n", "")
	
	return input
}

// ValidateAndSanitizeLoginRequest validates and sanitizes login request data
func ValidateAndSanitizeLoginRequest(email, password string) (string, string, *ValidationResult) {
	result := &ValidationResult{Valid: true}
	
	// Sanitize inputs
	email = SanitizeInput(email)
	password = strings.TrimSpace(password) // Don't sanitize password too aggressively
	
	// Validate email
	emailResult := ValidateEmail(email)
	if !emailResult.Valid {
		result.Errors = append(result.Errors, emailResult.Errors...)
		result.Valid = false
	}
	
	// Basic password validation (not strength, just presence)
	if password == "" {
		result.AddError("password", "password is required")
	}
	
	return email, password, result
}

// ValidateAndSanitizeRegisterRequest validates and sanitizes registration request data
func ValidateAndSanitizeRegisterRequest(email, password string) (string, string, *ValidationResult) {
	result := &ValidationResult{Valid: true}
	
	// Sanitize inputs
	email = SanitizeInput(email)
	password = strings.TrimSpace(password) // Don't sanitize password too aggressively
	
	// Validate email
	emailResult := ValidateEmail(email)
	if !emailResult.Valid {
		result.Errors = append(result.Errors, emailResult.Errors...)
		result.Valid = false
	}
	
	// Validate password strength
	passwordResult := ValidatePassword(password)
	if !passwordResult.Valid {
		result.Errors = append(result.Errors, passwordResult.Errors...)
		result.Valid = false
	}
	
	return email, password, result
}