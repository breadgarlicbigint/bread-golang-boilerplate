package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	activitySvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/activity/service"
)

// ActivityLogger auto-captures every HTTP request as an inbound log
// and the response as an outbound log, linked by X-Request-Id (correlationId).
func ActivityLogger(svc *activitySvc.ActivityService) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		correlationID := c.GetString("requestId")
		actorID := c.GetString(CtxUserID)
		if actorID == "" {
			actorID = "anonymous"
		}
		// CtxTenantID is a string constant defined in this package (auth.go exports it)
		tenantID := c.GetString("tenantId")

		svc.LogInbound(c.Request.Context(), activitySvc.ActivityLog{
			CorrelationID: correlationID,
			ActorID:       actorID,
			ActorType:     resolveActorType(actorID),
			TenantID:      tenantID,
			Action:        activitySvc.ActionOutAPIResponse,
			HTTPMethod:    c.Request.Method,
			HTTPPath:      c.FullPath(),
			IPAddress:     c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
		})

		c.Next()

		latency := time.Since(start).Milliseconds()
		svc.LogAPIResponse(c.Request.Context(), correlationID, actorID,
			c.Request.Method, c.FullPath(), c.Writer.Status(), latency)
	}
}

func resolveActorType(actorID string) string {
	if actorID == "anonymous" {
		return "anonymous"
	}
	return "user"
}
