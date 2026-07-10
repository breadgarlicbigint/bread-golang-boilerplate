package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	analyticsDTO "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/analytics/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

const cacheTTL = 1 * time.Hour

type AnalyticsSvc interface {
	UserRegistrations(ctx context.Context, start, end time.Time, granularity string) (*analyticsDTO.UserRegistrationStats, error)
	UserChurn(ctx context.Context, start, end time.Time) (*analyticsDTO.UserChurnStats, error)
	SignupMethodBreakdown(ctx context.Context) ([]analyticsDTO.SignupMethodBreakdown, error)
	BlockedUserTrend(ctx context.Context, start, end time.Time) ([]analyticsDTO.TimePoint, error)
	LoginFrequency(ctx context.Context, start, end time.Time, granularity string) ([]analyticsDTO.TimePoint, error)
	LoginMethodBreakdown(ctx context.Context, start, end time.Time) ([]analyticsDTO.LoginMethodBreakdown, error)
	LockoutStats(ctx context.Context, maxAttempts int) (float64, int64, error)
	PasskeyAdoption(ctx context.Context) (*analyticsDTO.PasskeyAdoptionStats, error)
	CredentialStuffingSignals(ctx context.Context, windowMin int, minAccounts int) ([]analyticsDTO.CredentialStuffingSignal, error)
	DeviceProliferationSignals(ctx context.Context) ([]analyticsDTO.DeviceProliferationSignal, error)
	FraudSignalSummary(ctx context.Context, maxAttempts int) (*analyticsDTO.FraudSummary, error)
	MobileVerificationStats(ctx context.Context) (*analyticsDTO.MobileVerificationStats, error)
}

type AnalyticsHandler struct {
	svc         AnalyticsSvc
	rdb         *redis.Client
	maxAttempts int
}

func New(svc AnalyticsSvc, rdb *redis.Client, maxAttempts int) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc, rdb: rdb, maxAttempts: maxAttempts}
}

// RegisterRoutes mounts all analytics endpoints under /admin/analytics.
func (h *AnalyticsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	a := rg.Group("/admin/analytics")
	a.GET("/users/registrations",          h.UserRegistrations)
	a.GET("/users/churn",                  h.UserChurn)
	a.GET("/users/signup-methods",         h.SignupMethods)
	a.GET("/users/blocked-trend",          h.BlockedTrend)
	a.GET("/auth/login-frequency",         h.LoginFrequency)
	a.GET("/auth/login-methods",           h.LoginMethods)
	a.GET("/auth/lockout",                 h.LockoutStats)
	a.GET("/passkeys/adoption",            h.PasskeyAdoption)
	a.GET("/mobile/verification",          h.MobileVerification)
	a.GET("/anomalies/credential-stuffing", h.CredentialStuffing)
	a.GET("/anomalies/device-proliferation", h.DeviceProliferation)
	a.GET("/fraud/signals",                h.FraudSignals)
}

func (h *AnalyticsHandler) UserRegistrations(c *gin.Context) {
	q, start, end, ok := bindDateRange(c)
	if !ok {
		return
	}
	h.cached(c, fmt.Sprintf("analytics:reg:%s:%s:%s", q.StartDate, q.EndDate, q.Granularity), func() (interface{}, error) {
		return h.svc.UserRegistrations(c.Request.Context(), start, end, q.Granularity)
	})
}

func (h *AnalyticsHandler) UserChurn(c *gin.Context) {
	_, start, end, ok := bindDateRange(c)
	if !ok {
		return
	}
	h.cached(c, fmt.Sprintf("analytics:churn:%s:%s", start.Format("2006-01-02"), end.Format("2006-01-02")), func() (interface{}, error) {
		return h.svc.UserChurn(c.Request.Context(), start, end)
	})
}

func (h *AnalyticsHandler) SignupMethods(c *gin.Context) {
	h.cached(c, "analytics:signup-methods", func() (interface{}, error) {
		return h.svc.SignupMethodBreakdown(c.Request.Context())
	})
}

func (h *AnalyticsHandler) BlockedTrend(c *gin.Context) {
	_, start, end, ok := bindDateRange(c)
	if !ok {
		return
	}
	h.cached(c, fmt.Sprintf("analytics:blocked:%s:%s", start.Format("2006-01-02"), end.Format("2006-01-02")), func() (interface{}, error) {
		return h.svc.BlockedUserTrend(c.Request.Context(), start, end)
	})
}

func (h *AnalyticsHandler) LoginFrequency(c *gin.Context) {
	q, start, end, ok := bindDateRange(c)
	if !ok {
		return
	}
	h.cached(c, fmt.Sprintf("analytics:logins:%s:%s:%s", q.StartDate, q.EndDate, q.Granularity), func() (interface{}, error) {
		return h.svc.LoginFrequency(c.Request.Context(), start, end, q.Granularity)
	})
}

func (h *AnalyticsHandler) LoginMethods(c *gin.Context) {
	_, start, end, ok := bindDateRange(c)
	if !ok {
		return
	}
	h.cached(c, fmt.Sprintf("analytics:login-methods:%s:%s", start.Format("2006-01-02"), end.Format("2006-01-02")), func() (interface{}, error) {
		return h.svc.LoginMethodBreakdown(c.Request.Context(), start, end)
	})
}

func (h *AnalyticsHandler) LockoutStats(c *gin.Context) {
	h.cached(c, "analytics:lockout", func() (interface{}, error) {
		rate, locked, err := h.svc.LockoutStats(c.Request.Context(), h.maxAttempts)
		if err != nil {
			return nil, err
		}
		return gin.H{"lockoutRate": rate, "lockedUsers": locked}, nil
	})
}

func (h *AnalyticsHandler) PasskeyAdoption(c *gin.Context) {
	h.cached(c, "analytics:passkey-adoption", func() (interface{}, error) {
		return h.svc.PasskeyAdoption(c.Request.Context())
	})
}

func (h *AnalyticsHandler) MobileVerification(c *gin.Context) {
	h.cached(c, "analytics:mobile-verification", func() (interface{}, error) {
		return h.svc.MobileVerificationStats(c.Request.Context())
	})
}

func (h *AnalyticsHandler) CredentialStuffing(c *gin.Context) {
	h.cachedTTL(c, "analytics:stuffing", 5*time.Minute, func() (interface{}, error) {
		return h.svc.CredentialStuffingSignals(c.Request.Context(), 5, 10)
	})
}

func (h *AnalyticsHandler) DeviceProliferation(c *gin.Context) {
	h.cachedTTL(c, "analytics:device-prolif", 15*time.Minute, func() (interface{}, error) {
		return h.svc.DeviceProliferationSignals(c.Request.Context())
	})
}

func (h *AnalyticsHandler) FraudSignals(c *gin.Context) {
	h.cachedTTL(c, "analytics:fraud", 10*time.Minute, func() (interface{}, error) {
		return h.svc.FraudSignalSummary(c.Request.Context(), h.maxAttempts)
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func bindDateRange(c *gin.Context) (analyticsDTO.DateRangeQuery, time.Time, time.Time, bool) {
	var q analyticsDTO.DateRangeQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.BadRequest(c, "startDate and endDate are required (YYYY-MM-DD)")
		return q, time.Time{}, time.Time{}, false
	}
	if q.Granularity == "" {
		q.Granularity = "day"
	}
	start, end, err := q.Dates()
	if err != nil {
		response.BadRequest(c, "Invalid date format — use YYYY-MM-DD")
		return q, time.Time{}, time.Time{}, false
	}
	return q, start, end, true
}

func (h *AnalyticsHandler) cached(c *gin.Context, key string, fn func() (interface{}, error)) {
	h.cachedTTL(c, key, cacheTTL, fn)
}

func (h *AnalyticsHandler) cachedTTL(c *gin.Context, key string, ttl time.Duration, fn func() (interface{}, error)) {
	ctx := c.Request.Context()

	if raw, err := h.rdb.Get(ctx, key).Bytes(); err == nil {
		var cached interface{}
		if json.Unmarshal(raw, &cached) == nil {
			c.Header("X-Cache", "HIT")
			response.OK(c, "Data fetched", cached)
			return
		}
	}

	data, err := fn()
	if err != nil {
		response.LogInternal(err, "analytics query failed")
		response.InternalServerError(c, "Analytics query failed")
		return
	}

	if b, err := json.Marshal(data); err == nil {
		_ = h.rdb.Set(ctx, key, b, ttl)
	}
	c.Header("X-Cache", "MISS")
	response.OK(c, "Data fetched", data)
}
