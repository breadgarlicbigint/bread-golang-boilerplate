package middleware

import (
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/gin-gonic/gin"
)

// PermWildcard grants every permission (held by an admin role). Compared against
// the flattened permission list from the JWT.
const PermWildcard = "*"

// RequirePermission gates a route on one or more per-action permissions
// (e.g. "users:write", "audit:read"). The caller must hold at least one of the
// listed permissions, or the wildcard "*".
//
// Authorization is server-side only — clients merely hide controls. Must run
// after AuthJWTAccess (which projects claims.Permissions into the context). A
// permission constant that is seeded and documented is NOT enforced until a
// route mounts this — grep for the middleware, not the constant.
func RequirePermission(perms ...string) gin.HandlerFunc {
	required := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		required[p] = struct{}{}
	}
	return func(c *gin.Context) {
		held := PermissionsFromCtx(c)
		for _, p := range held {
			if p == PermWildcard {
				c.Next()
				return
			}
			if _, ok := required[p]; ok {
				c.Next()
				return
			}
		}
		response.ForbiddenI18n(c, "auth.insufficientPermission")
	}
}

// RequireTenant blocks a tenantless (platform-scoped) user from tenant-scoped
// routes. Tenant users carry a non-empty tenantId; a platform user has none and
// belongs only on platform routes.
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		if TenantIDFromCtx(c) == "" {
			response.ForbiddenI18n(c, "auth.tenantScopeRequired")
			return
		}
		c.Next()
	}
}

// ── Context accessors ─────────────────────────────────────────────────────────

// TenantIDFromCtx returns the authoritative tenant id from the JWT ("" ⇒ a
// platform-scoped user). Tenant-scoped repositories MUST filter on this.
func TenantIDFromCtx(c *gin.Context) string {
	v, _ := c.Get(CtxTenantID)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// PermissionsFromCtx returns the caller's flattened permission list.
func PermissionsFromCtx(c *gin.Context) []string {
	v, _ := c.Get(CtxPermissions)
	if s, ok := v.([]string); ok {
		return s
	}
	return nil
}

// HasPermission reports whether the caller holds perm (or the wildcard). Handy
// inside a handler that conditionally allows an action rather than gating the
// whole route.
func HasPermission(c *gin.Context, perm string) bool {
	for _, p := range PermissionsFromCtx(c) {
		if p == PermWildcard || p == perm {
			return true
		}
	}
	return false
}

// UserIDFromCtx returns the authenticated user id (actor) for audit stamping.
func UserIDFromCtx(c *gin.Context) string {
	return c.GetString(CtxUserID)
}
