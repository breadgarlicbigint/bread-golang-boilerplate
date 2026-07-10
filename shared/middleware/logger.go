package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RequestID injects a unique request ID into the context and response headers.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("requestId", id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// Logger logs each request with zap structured logging. Requests that ended
// in an error response are logged at Warn (4xx) or Error (5xx) — instead of
// Info — so they stand out on the console.
func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.String("raw_query", c.Request.URL.RawQuery),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("ip", c.ClientIP()),
			zap.String("request_id", c.GetString("requestId")),
			zap.String("user_agent", c.Request.UserAgent()),
		}
		if errMsg := c.GetString("errorMessage"); errMsg != "" {
			fields = append(fields, zap.String("error", errMsg))
		}

		switch {
		case status >= 500:
			log.Error("request", fields...)
		case status >= 400:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// Recovery catches panics and returns a 500.
func Recovery(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.FullPath()),
					zap.String("request_id", c.GetString("requestId")),
				)
				c.AbortWithStatusJSON(500, gin.H{
					"statusCode": 500,
					"message":    "Internal server error",
					"requestId":  c.GetString("requestId"),
				})
			}
		}()
		c.Next()
	}
}
