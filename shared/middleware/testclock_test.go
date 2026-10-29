package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/clock"
)

func clockEngine(allowed bool) (*gin.Engine, *time.Time) {
	gin.SetMode(gin.TestMode)
	var seen time.Time
	r := gin.New()
	r.Use(TestClock(allowed))
	r.GET("/now", func(c *gin.Context) {
		seen = clock.Now(c.Request.Context())
		c.Status(http.StatusOK)
	})
	return r, &seen
}

func get(r *gin.Engine, header string) {
	req := httptest.NewRequest(http.MethodGet, "/now", nil)
	if header != "" {
		req.Header.Set(TestNowHeader, header)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func TestTestClock_HonouredOnlyWhenAllowed(t *testing.T) {
	want := time.Date(2030, 4, 1, 0, 0, 0, 0, time.UTC)

	// Allowed: the header overrides the clock.
	r, seen := clockEngine(true)
	get(r, want.Format(time.RFC3339))
	if !seen.Equal(want) {
		t.Errorf("allowed: clock = %v, want override %v", seen, want)
	}

	// Not allowed: the header is never read — the real clock is used.
	r, seen = clockEngine(false)
	before := time.Now()
	get(r, want.Format(time.RFC3339))
	if seen.Equal(want) {
		t.Error("disallowed: the header must not override the clock")
	}
	if seen.Before(before.Add(-time.Minute)) {
		t.Errorf("disallowed: expected ~real now, got %v", seen)
	}
}

func TestTestClock_MalformedHeaderIgnored(t *testing.T) {
	r, seen := clockEngine(true)
	before := time.Now()
	get(r, "not-a-timestamp")
	if seen.Before(before.Add(-time.Minute)) {
		t.Errorf("a malformed header must fall back to the real clock, got %v", seen)
	}
}
