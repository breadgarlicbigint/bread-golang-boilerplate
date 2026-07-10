package handler

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	mongo   *database.MongoDB
	rdb     *redis.Client
	version string
	startAt time.Time
}

func New(mongo *database.MongoDB, rdb *redis.Client, version string) *HealthHandler {
	return &HealthHandler{mongo: mongo, rdb: rdb, version: version, startAt: time.Now()}
}

// RegisterRoutes wires health endpoints.
func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", h.Health)
	rg.GET("/health/live", h.Live)
	rg.GET("/health/ready", h.Ready)
}

// Health godoc
// @Summary     Full health check
// @Tags        health
// @Produce     json
// @Success     200
// @Router      /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	mongoOK := h.mongo.Ping(ctx) == nil
	redisOK := h.rdb.Ping(ctx).Err() == nil

	status := "healthy"
	code := http.StatusOK
	if !mongoOK || !redisOK {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	c.JSON(code, gin.H{
		"status":  status,
		"version": h.version,
		"uptime":  time.Since(h.startAt).String(),
		"checks": gin.H{
			"mongo": boolStatus(mongoOK),
			"redis": boolStatus(redisOK),
		},
		"memory": gin.H{
			"alloc":    mem.Alloc / 1024 / 1024,
			"totalAlloc": mem.TotalAlloc / 1024 / 1024,
			"sys":      mem.Sys / 1024 / 1024,
			"unit":     "MB",
		},
	})
}

// Live is a simple liveness probe (always 200 if process is running).
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready checks all downstream dependencies.
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	if h.mongo.Ping(ctx) != nil || h.rdb.Ping(ctx).Err() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func boolStatus(ok bool) string {
	if ok {
		return "ok"
	}
	return "error"
}
