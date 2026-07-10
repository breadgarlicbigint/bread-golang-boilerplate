package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

type APIKeyValidator interface {
	Validate(ctx context.Context, rawKey string) (keyID string, err error)
}

const CtxAPIKeyID = "apiKeyId"

// APIKeyProtected reads the configured header and validates the key.
func APIKeyProtected(header string, validator APIKeyValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(header)
		if raw == "" {
			response.Unauthorized(c, "API key missing")
			return
		}
		keyID, err := validator.Validate(c.Request.Context(), raw)
		if err != nil {
			response.Unauthorized(c, "Invalid or inactive API key")
			return
		}
		c.Set(CtxAPIKeyID, keyID)
		c.Next()
	}
}
