package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

// RateLimiter implements a simple fixed-window counter using Redis.
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

		pipe := rl.rdb.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, rl.window)
		if _, err := pipe.Exec(ctx); err != nil {
			// If Redis is down, let requests through (fail-open)
			c.Next()
			return
		}

		count := incr.Val()
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
