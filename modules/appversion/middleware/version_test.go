package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
)

// alwaysRequire makes every check a force-update, so anything NOT exempt is
// blocked — which is exactly what makes the self-lock visible.
type alwaysRequire struct{}

func (alwaysRequire) Check(ctx context.Context, p service.Platform, v string) (*service.VersionCheckResponse, error) {
	return &service.VersionCheckResponse{Status: service.UpdateRequired, MinVersion: "2.0.0", ClientVersion: v}, nil
}

func versionEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(VersionCheck(alwaysRequire{}))
	h := func(c *gin.Context) { c.String(http.StatusOK, "ok") }
	r.GET("/v1/app-version/check", h)
	r.GET("/health/live", h)
	r.GET("/metrics", h)
	r.GET("/v1/users", h)
	return r
}

func getWithVersion(r *gin.Engine, path string) int {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(HeaderAppVersion, "1.0.0")
	req.Header.Set(HeaderPlatform, "android")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestVersionCheck_ExemptPathsAreReachableWhileBlocking(t *testing.T) {
	r := versionEngine()

	// A force-update client must still reach the check that explains it, and
	// liveness/metrics.
	for _, p := range []string{"/v1/app-version/check", "/health/live", "/metrics"} {
		if code := getWithVersion(r, p); code != http.StatusOK {
			t.Errorf("exempt %s: got %d, want 200 (the feature self-locks)", p, code)
		}
	}

	// A non-exempt route is still blocked with 426.
	if code := getWithVersion(r, "/v1/users"); code != http.StatusUpgradeRequired {
		t.Errorf("/v1/users: got %d, want 426", code)
	}
}

// The exemption matches the ROUTE path, never a query string — otherwise a
// client could tack ?x=/health onto any blocked route to slip past.
func TestVersionCheck_ExemptionIgnoresQueryString(t *testing.T) {
	if code := getWithVersion(versionEngine(), "/v1/users?next=/health"); code != http.StatusUpgradeRequired {
		t.Errorf("got %d, want 426 — a query string must not exempt a route", code)
	}
}

// A client sending no version headers is never blocked (web/browser clients).
func TestVersionCheck_NoHeadersPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(VersionCheck(alwaysRequire{}))
	r.GET("/v1/users", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("no version headers: got %d, want 200", w.Code)
	}
}
