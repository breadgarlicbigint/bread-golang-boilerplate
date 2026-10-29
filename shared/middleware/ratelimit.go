package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

// incrWithWindow atomically increments the counter and sets the window TTL ONLY
// when the key is first created (result == 1). Doing the EXPIRE on every request
// — as a plain INCR+EXPIRE pipeline does — pushes the window forward on each
// call, so a client that keeps hitting the limit never rolls out of the window
// and stays blocked indefinitely instead of being re-admitted after it elapses.
// A single script keeps the increment and the conditional expiry atomic so two
// concurrent first-requests cannot both skip the EXPIRE and leak a key with no
// TTL.
var incrWithWindow = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[1])
end
return count
`)

// RateLimiter implements a fixed-window counter using Redis.
type RateLimiter struct {
	rdb      *redis.Client
	max      int
	window   time.Duration
	keyScope string
}

func NewRateLimiter(rdb *redis.Client, max int, window time.Duration, scope string) *RateLimiter {
	return &RateLimiter{rdb: rdb, max: max, window: window, keyScope: scope}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := fmt.Sprintf("rl:%s:%s", rl.keyScope, c.ClientIP())
		ctx := context.Background()

		windowSecs := int(rl.window.Seconds())
		if windowSecs < 1 {
			windowSecs = 1
		}
		result, err := incrWithWindow.Run(ctx, rl.rdb, []string{key}, windowSecs).Result()
		if err != nil {
			// If Redis is down, let requests through (fail-open)
			c.Next()
			return
		}
		count, _ := result.(int64)
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.max))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", max64(int64(rl.max)-count, 0)))

		if count > int64(rl.max) {
			response.TooManyRequests(c)
			return
		}
		c.Next()
	}
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
