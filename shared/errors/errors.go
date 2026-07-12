package errors

import (
	"errors"
	"net/http"
)

// AppError is a structured domain error that carries an HTTP status.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// Key is the locales/*.json key used to translate Message for the request's
	// x-custom-lang. Empty for errors that don't have (or don't need) a
	// translation — callers fall back to the raw Message in that case.
	Key    string `json:"-"`
	Status int    `json:"-"`
	Err    error  `json:"-"`
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

// New creates a new AppError with no i18n key — Message is sent as-is
// regardless of x-custom-lang. Prefer NewI18n for anything user-facing.
func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// NewI18n creates a new AppError whose Message is translated via key when a
// handler renders it through response.HandleAppError / ErrorI18n.
func NewI18n(status int, code, key, message string) *AppError {
	return &AppError{Status: status, Code: code, Key: key, Message: message}
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
	ErrNotFound        = NewI18n(http.StatusNotFound, "NOT_FOUND", "http.404", "Resource not found")
	ErrUnauthorized    = NewI18n(http.StatusUnauthorized, "UNAUTHORIZED", "http.401", "Authentication required")
	ErrForbidden       = NewI18n(http.StatusForbidden, "FORBIDDEN", "http.403", "Access denied")
	ErrConflict        = NewI18n(http.StatusConflict, "CONFLICT", "http.409", "Resource already exists")
	ErrBadRequest      = NewI18n(http.StatusBadRequest, "BAD_REQUEST", "http.400", "Invalid request")
	ErrInternal        = NewI18n(http.StatusInternalServerError, "INTERNAL_ERROR", "http.500", "Internal server error")
	ErrTooManyRequests = NewI18n(http.StatusTooManyRequests, "TOO_MANY_REQUESTS", "http.429", "Rate limit exceeded")

	// Auth
	ErrInvalidCredentials = NewI18n(http.StatusUnauthorized, "INVALID_CREDENTIALS", "auth.invalidCredentials", "Invalid email or password")
	ErrAccountLocked      = NewI18n(http.StatusUnauthorized, "ACCOUNT_LOCKED", "auth.accountLocked", "Account is temporarily locked")
	ErrTokenExpired       = NewI18n(http.StatusUnauthorized, "TOKEN_EXPIRED", "auth.tokenExpired", "Token has expired")
	ErrTokenInvalid       = NewI18n(http.StatusUnauthorized, "TOKEN_INVALID", "auth.tokenInvalid", "Token is invalid")
	ErrSessionRevoked     = NewI18n(http.StatusUnauthorized, "SESSION_REVOKED", "auth.sessionRevoked", "Session has been revoked")
	ErrEmailNotVerified   = NewI18n(http.StatusForbidden, "EMAIL_NOT_VERIFIED", "auth.emailNotVerified", "Email address not verified")

	// User
	ErrUserNotFound      = NewI18n(http.StatusNotFound, "USER_NOT_FOUND", "user.notFound", "User not found")
	ErrEmailTaken        = NewI18n(http.StatusConflict, "EMAIL_TAKEN", "user.emailTaken", "Email is already registered")
	ErrUsernameTaken     = NewI18n(http.StatusConflict, "USERNAME_TAKEN", "user.usernameTaken", "Username is already taken")
	ErrInvalidPassword   = NewI18n(http.StatusBadRequest, "INVALID_PASSWORD", "user.invalidPassword", "Password does not meet requirements")
	ErrPasswordSameAsOld = NewI18n(http.StatusBadRequest, "PASSWORD_SAME_AS_OLD", "user.passwordSameAsOld", "New password must be different from the current password")

	// API Key
	ErrAPIKeyNotFound = NewI18n(http.StatusNotFound, "API_KEY_NOT_FOUND", "apiKey.notFound", "API key not found")
	ErrAPIKeyInvalid  = NewI18n(http.StatusUnauthorized, "API_KEY_INVALID", "apiKey.invalid", "API key is invalid or inactive")
	ErrAPIKeyExpired  = NewI18n(http.StatusUnauthorized, "API_KEY_EXPIRED", "apiKey.expired", "API key has expired")

	// Role
	ErrRoleNotFound = NewI18n(http.StatusNotFound, "ROLE_NOT_FOUND", "role.notFound", "Role not found")

	// Feature flag
	ErrFeatureDisabled = NewI18n(http.StatusForbidden, "FEATURE_DISABLED", "featureFlag.disabled", "This feature is not enabled")

	// Notification
	ErrMailerNotConfigured = NewI18n(http.StatusServiceUnavailable, "MAILER_NOT_CONFIGURED", "notification.mailerNotConfigured", "Email delivery is not configured")
	ErrEmailNotAvailable   = NewI18n(http.StatusBadRequest, "EMAIL_NOT_AVAILABLE", "notification.emailNotAvailable", "Recipient email address is required in data.email for the email channel")
)
