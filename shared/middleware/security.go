package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// SecurityConfig controls which headers are set.
type SecurityConfig struct {
	// Content-Security-Policy
	CSP string
	// Strict-Transport-Security max-age in seconds (0 = disabled)
	HSTSMaxAge int
	HSTSIncludeSubdomains bool
	HSTSPreload bool
	// Referrer-Policy
	ReferrerPolicy string
	// Permissions-Policy
	PermissionsPolicy string
	// X-Content-Type-Options: nosniff
	NoSniff bool
	// X-Frame-Options: DENY | SAMEORIGIN
	FrameOptions string
	// X-XSS-Protection (legacy)
	XSSProtection bool
	// Cross-Origin-Embedder-Policy
	COEP string
	// Cross-Origin-Opener-Policy
	COOP string
	// Cross-Origin-Resource-Policy
	CORP string
}

// DefaultSecurityConfig returns sensible defaults matching OWASP recommendations.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		CSP:                   "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none';",
		HSTSMaxAge:            31536000, // 1 year
		HSTSIncludeSubdomains: true,
		HSTSPreload:           false,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermissionsPolicy:     "camera=(), microphone=(), geolocation=(), payment=()",
		NoSniff:               true,
		FrameOptions:          "DENY",
		XSSProtection:         true,
		COEP:                  "require-corp",
		COOP:                  "same-origin",
		CORP:                  "same-origin",
	}
}

// APISecurityConfig is relaxed CSP for pure JSON APIs (no HTML rendered).
func APISecurityConfig() SecurityConfig {
	cfg := DefaultSecurityConfig()
	cfg.CSP = "" // APIs don't serve HTML; CSP is irrelevant
	cfg.FrameOptions = "DENY"
	cfg.COEP = ""
	cfg.COOP = ""
	cfg.CORP = ""
	return cfg
}

// SecurityHeaders sets all configured security headers on every response.
// Equivalent to Helmet.js in NestJS.
func SecurityHeaders(cfg SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()

		// X-Content-Type-Options
		if cfg.NoSniff {
			h.Set("X-Content-Type-Options", "nosniff")
		}

		// X-Frame-Options
		if cfg.FrameOptions != "" {
			h.Set("X-Frame-Options", cfg.FrameOptions)
		}

		// X-XSS-Protection (legacy browsers)
		if cfg.XSSProtection {
			h.Set("X-XSS-Protection", "1; mode=block")
		}

		// Strict-Transport-Security (HSTS)
		if cfg.HSTSMaxAge > 0 {
			hsts := fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
			if cfg.HSTSIncludeSubdomains {
				hsts += "; includeSubDomains"
			}
			if cfg.HSTSPreload {
				hsts += "; preload"
			}
			h.Set("Strict-Transport-Security", hsts)
		}

		// Content-Security-Policy
		if cfg.CSP != "" {
			h.Set("Content-Security-Policy", cfg.CSP)
		}

		// Referrer-Policy
		if cfg.ReferrerPolicy != "" {
			h.Set("Referrer-Policy", cfg.ReferrerPolicy)
		}

		// Permissions-Policy
		if cfg.PermissionsPolicy != "" {
			h.Set("Permissions-Policy", cfg.PermissionsPolicy)
		}

		// Cross-Origin-* policies
		if cfg.COEP != "" {
			h.Set("Cross-Origin-Embedder-Policy", cfg.COEP)
		}
		if cfg.COOP != "" {
			h.Set("Cross-Origin-Opener-Policy", cfg.COOP)
		}
		if cfg.CORP != "" {
			h.Set("Cross-Origin-Resource-Policy", cfg.CORP)
		}

		h.Set("X-DNS-Prefetch-Control", "off")

		c.Next()

		// Remove server fingerprint after downstream handlers run, in case one
		// of them sets X-Powered-By (e.g. a proxied/upstream header echo).
		h.Del("X-Powered-By")
	}
}
