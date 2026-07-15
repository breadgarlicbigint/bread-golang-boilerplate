package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGinMiddlewareRecordsRequestMetrics(t *testing.T) {
	engine := gin.New()
	engine.Use(GinMiddleware())
	engine.GET("/metrics-test/widgets/:id", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics-test/widgets/42", nil)
	engine.ServeHTTP(httptest.NewRecorder(), req)

	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, "/metrics-test/widgets/:id", "201"))
	if got != 1 {
		t.Fatalf("httpRequestsTotal = %v, want 1", got)
	}

	count := testutil.CollectAndCount(httpRequestDuration, "http_request_duration_seconds")
	if count == 0 {
		t.Fatal("expected http_request_duration_seconds to have observations")
	}
}

func TestGinMiddlewareFallsBackToUnmatchedForUnknownRoutes(t *testing.T) {
	engine := gin.New()
	engine.Use(GinMiddleware())
	// No routes registered — gin's NoRoute handler fires, so FullPath() is "".

	req := httptest.NewRequest(http.MethodGet, "/metrics-test/does-not-exist", nil)
	engine.ServeHTTP(httptest.NewRecorder(), req)

	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, "unmatched", "404"))
	if got < 1 {
		t.Fatalf("httpRequestsTotal[unmatched] = %v, want >= 1", got)
	}
}

func TestGinMiddlewareSkipsMetricsEndpointItself(t *testing.T) {
	engine := gin.New()
	engine.Use(GinMiddleware())
	engine.GET("/metrics", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	engine.ServeHTTP(httptest.NewRecorder(), req)

	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, "/metrics", "200"))
	if got != 0 {
		t.Fatalf("httpRequestsTotal[/metrics] = %v, want 0 (scrapes should not self-record)", got)
	}
}

func startedEvent(requestID int64, commandName, collection string) *event.CommandStartedEvent {
	raw, err := bson.Marshal(bson.D{{Key: commandName, Value: collection}})
	if err != nil {
		panic(err)
	}
	return &event.CommandStartedEvent{
		Command:     bson.Raw(raw),
		CommandName: commandName,
		RequestID:   requestID,
	}
}

func TestMongoCommandMonitorRecordsSuccess(t *testing.T) {
	monitor := MongoCommandMonitor(zap.NewNop())
	ctx := context.Background()

	monitor.Started(ctx, startedEvent(1001, "find", "widgets_success"))
	monitor.Succeeded(ctx, &event.CommandSucceededEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{
			CommandName: "find",
			RequestID:   1001,
			Duration:    5 * time.Millisecond,
		},
	})

	gotCount := testutil.ToFloat64(mongoOperationsTotal.WithLabelValues("find", "widgets_success", "success"))
	if gotCount != 1 {
		t.Fatalf("mongoOperationsTotal = %v, want 1", gotCount)
	}
	obs := testutil.CollectAndCount(mongoOperationDuration, "mongodb_operation_duration_seconds")
	if obs == 0 {
		t.Fatal("expected mongodb_operation_duration_seconds to have observations")
	}
}

func TestMongoCommandMonitorRecordsFailure(t *testing.T) {
	monitor := MongoCommandMonitor(zap.NewNop())
	ctx := context.Background()

	monitor.Started(ctx, startedEvent(1002, "insert", "widgets_failure"))
	monitor.Failed(ctx, &event.CommandFailedEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{
			CommandName: "insert",
			RequestID:   1002,
			Duration:    2 * time.Millisecond,
		},
		Failure: "duplicate key error",
	})

	gotCount := testutil.ToFloat64(mongoOperationsTotal.WithLabelValues("insert", "widgets_failure", "error"))
	if gotCount != 1 {
		t.Fatalf("mongoOperationsTotal = %v, want 1", gotCount)
	}
}

func TestMongoCommandMonitorSkipsHeartbeatCommands(t *testing.T) {
	monitor := MongoCommandMonitor(zap.NewNop())
	ctx := context.Background()

	monitor.Started(ctx, startedEvent(1003, "ismaster", "n/a"))
	monitor.Succeeded(ctx, &event.CommandSucceededEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{
			CommandName: "ismaster",
			RequestID:   1003,
			Duration:    time.Millisecond,
		},
	})

	got := testutil.ToFloat64(mongoOperationsTotal.WithLabelValues("ismaster", "", "success"))
	if got != 0 {
		t.Fatalf("mongoOperationsTotal[ismaster] = %v, want 0 (heartbeat commands should be filtered)", got)
	}
}

func TestMongoCommandMonitorIgnoresSucceededWithoutMatchingStarted(t *testing.T) {
	monitor := MongoCommandMonitor(zap.NewNop())
	ctx := context.Background()

	// No Started call for RequestID 9999 — simulates a dropped/out-of-order event.
	monitor.Succeeded(ctx, &event.CommandSucceededEvent{
		CommandFinishedEvent: event.CommandFinishedEvent{
			CommandName: "aggregate_orphan",
			RequestID:   9999,
			Duration:    time.Millisecond,
		},
	})

	got := testutil.ToFloat64(mongoOperationsTotal.WithLabelValues("aggregate_orphan", "", "success"))
	if got != 0 {
		t.Fatalf("mongoOperationsTotal[aggregate_orphan] = %v, want 0 (no matching Started event)", got)
	}
}
