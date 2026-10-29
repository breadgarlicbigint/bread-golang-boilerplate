package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/clock"
)

// TestNowHeader carries an RFC3339 instant that overrides the server clock for
// one request.
const TestNowHeader = "X-Test-Now"

// TestClock honours TestNowHeader — but ONLY when `allowed` is true, which comes
// from ALLOW_TEST_CLOCK and is false in the default config and in every deployed
// environment. It lets a live suite exercise date boundaries without restarting
// the server against a frozen clock.
//
// Three properties, each deliberate:
//
//   - With `allowed` false this returns a middleware that never reads the header
//     at all, rather than one that reads and ignores it. There is no code path
//     in a production binary that parses caller-supplied time.
//   - A malformed value is IGNORED, not refused. The header is a test
//     affordance; failing a real request because a proxy injected something odd
//     into a header the app does not otherwise use is a worse outcome than using
//     the real clock.
//   - It sets the override on the request context only, so it lives exactly as
//     long as the request — a background worker with no request context is out
//     of this header's reach by construction, not by a check to remember.
func TestClock(allowed bool) gin.HandlerFunc {
	if !allowed {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		if raw := c.GetHeader(TestNowHeader); raw != "" {
			if t, err := time.Parse(time.RFC3339, raw); err == nil {
				c.Request = c.Request.WithContext(clock.WithNow(c.Request.Context(), t))
			}
		}
		c.Next()
	}
}
