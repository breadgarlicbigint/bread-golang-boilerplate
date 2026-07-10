package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// logger is set once at startup via SetLogger so error responses and
// unexpected internal errors are always visible on the console, even though
// the client only ever sees the safe envelope message.
var logger *zap.Logger

// SetLogger wires the application logger. Call once during app startup.
func SetLogger(l *zap.Logger) { logger = l }

// LogInternal prints an unexpected error to the console with full detail.
// Use it in handleError fallbacks before returning a generic message to the client.
func LogInternal(err error, msg string) {
	if logger == nil || err == nil {
		return
	}
	logger.Error(msg, zap.Error(err))
}

// Envelope is the standard JSON wrapper for every response.
type Envelope struct {
	StatusCode int         `json:"statusCode"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data,omitempty"`
	Meta       *Meta       `json:"_metadata,omitempty"`
	Timestamp  string      `json:"timestamp"`
	Path       string      `json:"path"`
	RequestID  string      `json:"requestId"`
}

// Meta holds pagination metadata.
type Meta struct {
	Total     int64  `json:"total"`
	Page      int    `json:"page"`
	PerPage   int    `json:"perPage"`
	TotalPage int    `json:"totalPage"`
	HasNext   bool   `json:"hasNext"`
	HasPrev   bool   `json:"hasPrev"`
	Cursor    string `json:"cursor,omitempty"`
}

// ErrorDetail is included in error responses.
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// ErrorEnvelope wraps errors.
type ErrorEnvelope struct {
	StatusCode int           `json:"statusCode"`
	Message    string        `json:"message"`
	Errors     []ErrorDetail `json:"errors,omitempty"`
	Timestamp  string        `json:"timestamp"`
	Path       string        `json:"path"`
	RequestID  string        `json:"requestId"`
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func requestID(c *gin.Context) string {
	if id := c.GetString("requestId"); id != "" {
		return id
	}
	return c.GetHeader("X-Request-Id")
}

// ── Success helpers ───────────────────────────────────────────────────────────

func OK(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Envelope{
		StatusCode: http.StatusOK,
		Message:    message,
		Data:       data,
		Timestamp:  now(),
		Path:       c.FullPath(),
		RequestID:  requestID(c),
	})
}

func OKWithMeta(c *gin.Context, message string, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Envelope{
		StatusCode: http.StatusOK,
		Message:    message,
		Data:       data,
		Meta:       meta,
		Timestamp:  now(),
		Path:       c.FullPath(),
		RequestID:  requestID(c),
	})
}

func Created(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{
		StatusCode: http.StatusCreated,
		Message:    message,
		Data:       data,
		Timestamp:  now(),
		Path:       c.FullPath(),
		RequestID:  requestID(c),
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// ── Error helpers ─────────────────────────────────────────────────────────────

func Error(c *gin.Context, status int, message string, errs ...ErrorDetail) {
	c.Set("errorMessage", message)
	c.AbortWithStatusJSON(status, ErrorEnvelope{
		StatusCode: status,
		Message:    message,
		Errors:     errs,
		Timestamp:  now(),
		Path:       c.FullPath(),
		RequestID:  requestID(c),
	})
}

func BadRequest(c *gin.Context, message string, errs ...ErrorDetail) {
	Error(c, http.StatusBadRequest, message, errs...)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message)
}

func UnprocessableEntity(c *gin.Context, message string, errs ...ErrorDetail) {
	Error(c, http.StatusUnprocessableEntity, message, errs...)
}

func TooManyRequests(c *gin.Context) {
	Error(c, http.StatusTooManyRequests, "Too many requests. Please slow down.")
}

func InternalServerError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
