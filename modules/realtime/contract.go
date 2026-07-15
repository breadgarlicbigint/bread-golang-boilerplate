// Package realtime provides live event delivery to connected clients over
// WebSocket and Server-Sent Events, plus a generic topic-based pub/sub
// primitive on top of the same connection pool.
//
// # Monolith usage
//
// Import the concrete service directly from modules/realtime/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers
// stay the same. Note that WebSocket/SSE connections themselves are
// inherently sticky to whichever instance holds them; a real extraction
// would need a shared broker (e.g. Redis pub/sub) fanning Publish calls out
// to every instance's local Hub, not just this instance's.
package realtime

// Service is the public contract other modules use to push a live event to
// connected WebSocket/SSE clients. Fire-and-forget: if nobody is currently
// connected to the target user/topic, the event is silently dropped — there
// is no buffering or replay. That's what GET /v1/me/notifications and
// GET /v1/admin/iot/telemetry are for; realtime delivery is a "while you're
// watching" convenience on top of those, not a source of truth.
type Service interface {
	// PublishToUser delivers evt to userID's private channel — every
	// GET /v1/me/ws and GET /v1/me/events connection for that user is
	// auto-subscribed to it. Returns the number of connections it reached.
	PublishToUser(userID string, evt Event) (int, error)
	// PublishToTopic delivers evt to every client subscribed to an
	// arbitrary topic — the generic pub/sub path exercised by
	// POST /v1/admin/realtime/publish. Returns the number of connections
	// it reached.
	PublishToTopic(topic string, evt Event) (int, error)
	// Stats reports current connection/topic occupancy for admin
	// diagnostics (GET /v1/admin/realtime/stats).
	Stats() Stats
}

// Event is a message delivered over WebSocket/SSE.
type Event struct {
	Type  string         `json:"type"`
	Title string         `json:"title,omitempty"`
	Body  string         `json:"body,omitempty"`
	Data  map[string]any `json:"data,omitempty"`
}

// Stats mirrors pkg/realtime.Stats as its own type so callers of this
// contract don't need to import pkg/realtime directly.
type Stats struct {
	TopicCount  int            `json:"topicCount"`
	ClientCount int            `json:"clientCount"`
	Topics      map[string]int `json:"topics"`
}
