package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

type FlagChecker interface {
	IsEnabled(ctx context.Context, key string) (bool, error)
}

// FeatureFlagProtected blocks the route if the named flag is disabled.
func FeatureFlagProtected(checker FlagChecker, flagKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabled, err := checker.IsEnabled(c.Request.Context(), flagKey)
		if err != nil || !enabled {
			response.ForbiddenI18n(c, "featureFlag.disabled")
			return
		}
		c.Next()
	}
}
