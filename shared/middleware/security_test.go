package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
)

func TestSecurityHeaders_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.SecurityHeaders(middleware.DefaultSecurityConfig()))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	checks := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"X-XSS-Protection":        "1; mode=block",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
	}
	for header, want := range checks {
		got := w.Header().Get(header)
		if got != want {
			t.Errorf("header %q: want %q, got %q", header, want, got)
		}
	}
}

func TestSecurityHeaders_HSTS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := middleware.DefaultSecurityConfig()
	cfg.HSTSMaxAge = 31536000
	cfg.HSTSIncludeSubdomains = true
	r.Use(middleware.SecurityHeaders(cfg))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("expected Strict-Transport-Security header")
	}
	if hsts != "max-age=31536000; includeSubDomains" {
		t.Errorf("unexpected HSTS value: %q", hsts)
	}
}

func TestSecurityHeaders_NoCSP_ForAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.SecurityHeaders(middleware.APISecurityConfig()))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if csp := w.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("API config should not set CSP, got: %q", csp)
	}
}

func TestSecurityHeaders_RemovesServerFingerprint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.SecurityHeaders(middleware.APISecurityConfig()))
	r.GET("/test", func(c *gin.Context) {
		c.Header("X-Powered-By", "Express")
		c.Status(200)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if powered := w.Header().Get("X-Powered-By"); powered != "" {
		t.Errorf("X-Powered-By should be removed, got: %q", powered)
	}
}

func TestSecurityHeaders_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Zero-value config — no HSTS, no CSP
	r.Use(middleware.SecurityHeaders(middleware.SecurityConfig{NoSniff: true}))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if hsts := w.Header().Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS should not be set when maxAge=0, got: %q", hsts)
	}
}
