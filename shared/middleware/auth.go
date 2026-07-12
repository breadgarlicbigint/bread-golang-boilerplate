package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	jwtpkg "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/jwt"
)

type SessionStore interface {
	Exists(ctx context.Context, sessionID string) (bool, error)
}

const (
	CtxUserID    = "userID"
	CtxSessionID = "sessionID"
	CtxRole      = "userRole"
	CtxClaims    = "jwtClaims"
)

// AuthJWTAccess validates the Bearer access token.
// It also checks the session is still active in Redis (stateful).
func AuthJWTAccess(jwtMgr *jwtpkg.Manager, sessions SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			response.UnauthorizedI18n(c, "auth.missingAuthHeader")
			return
		}
		tokenStr := strings.TrimPrefix(raw, "Bearer ")

		claims, err := jwtMgr.ParseAccess(tokenStr)
		if err != nil {
			response.UnauthorizedI18n(c, "auth.invalidAccessToken")
			return
		}

		// Stateful check: session must still be in Redis
		ok, err := sessions.Exists(c.Request.Context(), claims.SessionID)
		if err != nil || !ok {
			response.UnauthorizedI18n(c, "auth.sessionRevoked")
			return
		}

		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxSessionID, claims.SessionID)
		c.Set(CtxRole, claims.Role)
		c.Set(CtxClaims, claims)
		c.Next()
	}
}

// RoleProtected allows only the listed roles to proceed.
func RoleProtected(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get(CtxRole)
		if _, ok := allowed[role.(string)]; !ok {
			response.ForbiddenI18n(c, "auth.insufficientRole")
			return
		}
		c.Next()
	}
}

// MustBeOwnerOrAdmin verifies the caller owns the resource or has admin role.
func MustBeOwnerOrAdmin(paramName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		callerID := c.GetString(CtxUserID)
		callerRole := c.GetString(CtxRole)
		resourceOwner := c.Param(paramName)
		if callerRole != "admin" && callerID != resourceOwner {
			response.ForbiddenI18n(c, "auth.ownResourceOnly")
			return
		}
		c.Next()
	}
}
