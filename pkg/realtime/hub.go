// Package realtime is a transport-agnostic pub/sub fan-out core shared by
// WebSocket and SSE delivery. It has zero domain knowledge (no notion of
// users, notifications, or devices) — modules/realtime wraps Hub with those
// conventions (e.g. a per-user private topic).
package realtime

import (
	"sync"
	"time"
)

// Event is a single message fanned out to subscribers of a topic.
type Event struct {
	Topic     string    `json:"topic"`
	Type      string    `json:"type"`
	Title     string    `json:"title,omitempty"`
	Body      string    `json:"body,omitempty"`
	Data      any       `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Client is anything that can receive fanned-out events — a WebSocket
// connection, an SSE stream, or (in tests) an in-memory channel. Hub only
// ever calls Send; the handler package owns the actual transport.
type Client interface {
	ID() string
	// Send delivers evt to the client. It returns false when the client's
	// outbound buffer is full or already closed — the Hub treats that as
	// "slow or dead consumer" and drops it from the topic.
	Send(evt Event) bool
}

// Hub fans Events out to Clients subscribed to a topic.
type Hub struct {
	mu     sync.RWMutex
	topics map[string]map[string]Client // topic -> clientID -> Client
}

func NewHub() *Hub {
	return &Hub{topics: make(map[string]map[string]Client)}
}

// Subscribe adds c to topic. Safe to call multiple times (idempotent).
func (h *Hub) Subscribe(topic string, c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[string]Client)
	}
	h.topics[topic][c.ID()] = c
}

// Unsubscribe removes c from topic only.
func (h *Hub) Unsubscribe(topic string, c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unsubscribeLocked(topic, c.ID())
}

// UnsubscribeAll removes c from every topic it's subscribed to — call this
// once when a connection closes.
func (h *Hub) UnsubscribeAll(c Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := c.ID()
	for topic, clients := range h.topics {
		if _, ok := clients[id]; ok {
			h.unsubscribeLocked(topic, id)
		}
	}
}

func (h *Hub) unsubscribeLocked(topic, clientID string) {
	clients, ok := h.topics[topic]
	if !ok {
		return
	}
	delete(clients, clientID)
	if len(clients) == 0 {
		delete(h.topics, topic)
	}
}

// Publish fans evt out to every client currently subscribed to topic and
// returns how many received it. Clients whose Send returns false are
// dropped from the topic as a side effect (best-effort dead-connection
// cleanup — the transport layer still owns actually closing the socket).
func (h *Hub) Publish(topic string, evt Event) int {
	evt.Topic = topic
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}

	h.mu.RLock()
	clients := make([]Client, 0, len(h.topics[topic]))
	for _, c := range h.topics[topic] {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	delivered := 0
	var dead []Client
	for _, c := range clients {
		if c.Send(evt) {
			delivered++
		} else {
			dead = append(dead, c)
		}
	}
	if len(dead) > 0 {
		h.mu.Lock()
		for _, c := range dead {
			h.unsubscribeLocked(topic, c.ID())
		}
		h.mu.Unlock()
	}
	return delivered
}

// Stats summarizes current hub occupancy — used by the admin diagnostic
// endpoint (GET /v1/admin/realtime/stats) so the routing behavior below is
// actually observable from the web test client, not just inferred from logs.
type Stats struct {
	TopicCount  int            `json:"topicCount"`
	ClientCount int            `json:"clientCount"` // distinct clients across all topics
	Topics      map[string]int `json:"topics"`       // topic -> subscriber count
}

func (h *Hub) Stats() Stats {
	h.mu.RLock()
	defer h.mu.RUnlock()
	topics := make(map[string]int, len(h.topics))
	seen := make(map[string]struct{})
	for topic, clients := range h.topics {
		topics[topic] = len(clients)
		for id := range clients {
			seen[id] = struct{}{}
		}
	}
	return Stats{TopicCount: len(topics), ClientCount: len(seen), Topics: topics}
}
