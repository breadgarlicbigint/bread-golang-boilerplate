package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
)

// Headers clients must send with every request.
const (
	HeaderAppVersion = "X-App-Version"  // e.g. "2.4.1"
	HeaderPlatform   = "X-App-Platform" // "ios" | "android" | "web"
)

type VersionChecker interface {
	Check(ctx context.Context, platform service.Platform, clientVersion string) (*service.VersionCheckResponse, error)
}

// VersionCheck inspects X-App-Version + X-App-Platform headers and:
//   - Blocks the request (503) when a force update is required
//   - Injects the version check result into the context for handlers to surface
//   - Passes through when headers are absent (web/browser clients)
func VersionCheck(checker VersionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawVersion := c.GetHeader(HeaderAppVersion)
		rawPlatform := strings.ToLower(c.GetHeader(HeaderPlatform))

		// Not a versioned client — skip
		if rawVersion == "" || rawPlatform == "" {
			c.Next()
			return
		}

		platform := service.Platform(rawPlatform)
		result, err := checker.Check(c.Request.Context(), platform, rawVersion)
		if err != nil {
			// Non-blocking — don't fail requests because of version DB errors
			c.Next()
			return
		}

		// Always surface version info in response headers so clients can self-update
		c.Header("X-Version-Status", string(result.Status))
		c.Header("X-Current-Version", result.CurrentVersion)
		c.Header("X-Min-Version", result.MinVersion)

		// Hard block on force-update
		if result.Status == service.UpdateRequired {
			c.AbortWithStatusJSON(426, gin.H{
				"statusCode":     426,
				"message":        "Application update required. Please update to continue.",
				"status":         result.Status,
				"currentVersion": result.CurrentVersion,
				"minVersion":     result.MinVersion,
				"clientVersion":  result.ClientVersion,
				"storeUrl":       result.StoreURL,
				"releaseNotes":   result.ReleaseNotes,
				"forceUpdate":    true,
			})
			return
		}

		// Store for handlers/analytics
		c.Set("versionStatus", string(result.Status))
		c.Set("clientVersion", rawVersion)
		c.Set("clientPlatform", rawPlatform)
		c.Next()
	}
}
