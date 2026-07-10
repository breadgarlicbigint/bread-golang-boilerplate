package errors

import (
	"errors"
	"net/http"
)

// AppError is a structured domain error that carries an HTTP status.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// New creates a new AppError.
func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// Wrap wraps an underlying error.
func Wrap(status int, code, message string, err error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Err: err}
}

// As unwraps to *AppError.
func As(err error) (*AppError, bool) {
	var ae *AppError
	return ae, errors.As(err, &ae)
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var (
	ErrNotFound        = New(http.StatusNotFound, "NOT_FOUND", "Resource not found")
	ErrUnauthorized    = New(http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
	ErrForbidden       = New(http.StatusForbidden, "FORBIDDEN", "Access denied")
	ErrConflict        = New(http.StatusConflict, "CONFLICT", "Resource already exists")
	ErrBadRequest      = New(http.StatusBadRequest, "BAD_REQUEST", "Invalid request")
	ErrInternal        = New(http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	ErrTooManyRequests = New(http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "Rate limit exceeded")

	// Auth
	ErrInvalidCredentials = New(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	ErrAccountLocked      = New(http.StatusUnauthorized, "ACCOUNT_LOCKED", "Account is temporarily locked")
	ErrTokenExpired       = New(http.StatusUnauthorized, "TOKEN_EXPIRED", "Token has expired")
	ErrTokenInvalid       = New(http.StatusUnauthorized, "TOKEN_INVALID", "Token is invalid")
	ErrSessionRevoked     = New(http.StatusUnauthorized, "SESSION_REVOKED", "Session has been revoked")
	ErrEmailNotVerified   = New(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Email address not verified")

	// User
	ErrUserNotFound      = New(http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	ErrEmailTaken        = New(http.StatusConflict, "EMAIL_TAKEN", "Email is already registered")
	ErrUsernameTaken     = New(http.StatusConflict, "USERNAME_TAKEN", "Username is already taken")
	ErrInvalidPassword   = New(http.StatusBadRequest, "INVALID_PASSWORD", "Password does not meet requirements")
	ErrPasswordSameAsOld = New(http.StatusBadRequest, "PASSWORD_SAME_AS_OLD", "New password must be different from the current password")

	// API Key
	ErrAPIKeyNotFound = New(http.StatusNotFound, "API_KEY_NOT_FOUND", "API key not found")
	ErrAPIKeyInvalid  = New(http.StatusUnauthorized, "API_KEY_INVALID", "API key is invalid or inactive")
	ErrAPIKeyExpired  = New(http.StatusUnauthorized, "API_KEY_EXPIRED", "API key has expired")

	// Role
	ErrRoleNotFound = New(http.StatusNotFound, "ROLE_NOT_FOUND", "Role not found")

	// Feature flag
	ErrFeatureDisabled = New(http.StatusForbidden, "FEATURE_DISABLED", "This feature is not enabled")
)
