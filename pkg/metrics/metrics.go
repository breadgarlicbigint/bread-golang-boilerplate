// Package metrics wraps prometheus/client_golang for the whole app: an
// HTTP request Gin middleware, a MongoDB driver CommandMonitor (metrics +
// zap logging), and the /metrics scrape handler. All metrics register on
// the default Prometheus registry, so a single GET /metrics exposes Go
// runtime and process stats (via promauto's default registerer) alongside
// these application metrics.
package metrics

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/event"
	"go.uber.org/zap"
)

const metricsPath = "/metrics"

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed, labeled by method, route, and status code.",
	}, []string{"method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds, labeled by method, route, and status code.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	mongoOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mongodb_operations_total",
		Help: "Total MongoDB commands executed, labeled by operation, collection, and outcome.",
	}, []string{"operation", "collection", "status"})

	mongoOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "mongodb_operation_duration_seconds",
		Help:    "MongoDB command latency in seconds, labeled by operation and collection.",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation", "collection"})
)

// Handler serves the Prometheus text exposition format for scraping.
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return gin.WrapH(h)
}

// GinMiddleware records request count and latency for every request except
// the /metrics endpoint itself, so scrapes don't pollute their own series.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		if c.Request.URL.Path == metricsPath {
			return
		}

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(duration)
	}
}

// heartbeatCommands are high-frequency driver/admin chatter (connection
// handshakes, session keepalives) that would otherwise dominate both the
// log output and the mongodb_operations_total cardinality without carrying
// any signal about actual application queries.
var heartbeatCommands = map[string]bool{
	"hello":        true,
	"ismaster":     true,
	"isMaster":     true,
	"ping":         true,
	"saslStart":    true,
	"saslContinue": true,
	"endSessions":  true,
	"buildInfo":    true,
	"getLastError": true,
}

type commandStart struct {
	collection string
}

// MongoCommandMonitor returns an *event.CommandMonitor that logs every
// non-heartbeat MongoDB command via zap (operation, collection, duration,
// outcome) and records mongodb_operations_total /
// mongodb_operation_duration_seconds. Pass it to
// database.NewMongoDBWithMonitor. The collection name is only available on
// the Started event's raw command document (e.g. {"find": "users", ...}),
// so it's captured there and looked up again by RequestID when the
// Succeeded/Failed event arrives — those events carry Duration and
// CommandName directly but not the original command document.
func MongoCommandMonitor(log *zap.Logger) *event.CommandMonitor {
	var mu sync.Mutex
	inFlight := make(map[int64]commandStart)

	return &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			if heartbeatCommands[evt.CommandName] {
				return
			}
			collection := ""
			if v, err := evt.Command.LookupErr(evt.CommandName); err == nil {
				if s, ok := v.StringValueOK(); ok {
					collection = s
				}
			}
			mu.Lock()
			inFlight[evt.RequestID] = commandStart{collection: collection}
			mu.Unlock()
		},
		Succeeded: func(_ context.Context, evt *event.CommandSucceededEvent) {
			if heartbeatCommands[evt.CommandName] {
				return
			}
			mu.Lock()
			start, ok := inFlight[evt.RequestID]
			delete(inFlight, evt.RequestID)
			mu.Unlock()
			if !ok {
				return
			}

			mongoOperationsTotal.WithLabelValues(evt.CommandName, start.collection, "success").Inc()
			mongoOperationDuration.WithLabelValues(evt.CommandName, start.collection).Observe(evt.Duration.Seconds())
			log.Debug("mongodb command",
				zap.String("operation", evt.CommandName),
				zap.String("collection", start.collection),
				zap.Duration("duration", evt.Duration),
			)
		},
		Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
			if heartbeatCommands[evt.CommandName] {
				return
			}
			mu.Lock()
			start, ok := inFlight[evt.RequestID]
			delete(inFlight, evt.RequestID)
			mu.Unlock()
			if !ok {
				return
			}

			mongoOperationsTotal.WithLabelValues(evt.CommandName, start.collection, "error").Inc()
			mongoOperationDuration.WithLabelValues(evt.CommandName, start.collection).Observe(evt.Duration.Seconds())
			log.Warn("mongodb command failed",
				zap.String("operation", evt.CommandName),
				zap.String("collection", start.collection),
				zap.Duration("duration", evt.Duration),
				zap.String("failure", evt.Failure),
			)
		},
	}
}
