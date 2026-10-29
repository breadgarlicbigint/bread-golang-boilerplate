package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/audit"
)

// headerAppPlatform mirrors modules/appversion/middleware.HeaderPlatform.
// Repeated here rather than imported so shared/ keeps depending on nothing in
// modules/.
const headerAppPlatform = "X-App-Platform"

// AuditContext puts the request's actor facts (tenant, user, role, client
// platform, IP, correlation id) on the request context, where any service can
// read them when writing an audit entry — without threading them through every
// method signature, and without ever taking them from a request body.
//
// It registers a **lazy provider** rather than a snapshot: this middleware is
// global, while the auth middleware that sets the claims runs later, per route
// group. Reading the gin context only when an entry is written makes the actor
// available regardless of middleware ordering.
func AuditContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		provider := func() audit.RequestInfo {
			tenantID, _ := uuid.Parse(TenantIDFromCtx(c))
			return audit.RequestInfo{
				TenantID:       tenantID,
				ActorID:        UserIDFromCtx(c),
				ActorRole:      c.GetString(CtxRole),
				ClientPlatform: strings.ToLower(c.GetHeader(headerAppPlatform)),
				IPAddress:      c.ClientIP(),
				// Same correlation id as the HTTP activity log, so an audit entry
				// and the request that caused it can be joined.
				CorrelationID: c.GetString("requestId"),
			}
		}
		c.Request = c.Request.WithContext(audit.WithProvider(c.Request.Context(), provider))
		c.Next()
	}
}
