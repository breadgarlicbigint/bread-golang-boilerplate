package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func init() { gin.SetMode(gin.TestMode) }

// newLimiterStack wires a RateLimiter over an in-memory Redis and returns the
// engine plus the fake clock so a test can advance time past the window.
func newLimiterStack(t *testing.T, max int, window time.Duration) (*gin.Engine, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	rl := NewRateLimiter(rdb, max, window, "test")
	r := gin.New()
	r.Use(rl.Middleware())
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r, mr
}

func do(r *gin.Engine) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// The regression guard for the sliding-window bug: a client blocked at the limit
// must be admitted again once the window has elapsed. The old INCR+EXPIRE-every-
// request code renewed the TTL on each call, so this client stayed at 429 forever.
func TestRateLimiter_ReadmitsAfterWindowElapses(t *testing.T) {
	const max = 3
	const window = time.Minute
	r, mr := newLimiterStack(t, max, window)

	for i := 1; i <= max; i++ {
		if code := do(r); code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, code)
		}
	}
	// The next one is over the limit.
	if code := do(r); code != http.StatusTooManyRequests {
		t.Fatalf("over-limit request: got %d, want 429", code)
	}

	// Keep hammering while still inside the window — this is what used to renew
	// the TTL and trap the client.
	for i := 0; i < max; i++ {
		if code := do(r); code != http.StatusTooManyRequests {
			t.Fatalf("still-blocked request %d: got %d, want 429", i, code)
		}
	}

	// Advance past the window; the key must have expired and the client is let
	// back in.
	mr.FastForward(window + time.Second)
	if code := do(r); code != http.StatusOK {
		t.Fatalf("after window elapsed: got %d, want 200 (client never re-admitted — the fixed window is sliding)", code)
	}
}

// Within a single window the limiter admits exactly max requests then blocks.
func TestRateLimiter_BlocksOverTheLimitWithinWindow(t *testing.T) {
	const max = 5
	r, _ := newLimiterStack(t, max, time.Minute)
	for i := 1; i <= max; i++ {
		if code := do(r); code != http.StatusOK {
			t.Fatalf("request %d: got %d, want 200", i, code)
		}
	}
	if code := do(r); code != http.StatusTooManyRequests {
		t.Fatalf("request %d: got %d, want 429", max+1, code)
	}
}
