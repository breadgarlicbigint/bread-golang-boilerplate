// Package queue defines a backend-agnostic job/worker contract shared by every
// queue transport in this project (RabbitMQ, Kafka, …). It mirrors the shape of
// the Redis/Asynq implementation in pkg/worker so the three systems are drop-in
// parallel:
//
//	pkg/worker            Redis (Asynq)     — Client / Server / HandlerFunc(*asynq.Task)
//	pkg/queue/rabbitmq     RabbitMQ (AMQP)   — Publisher / Consumer / HandlerFunc(Delivery)
//	pkg/queue/kafka        Kafka             — Publisher / Consumer / HandlerFunc(Delivery)
//
// A "job" is produced through a Publisher; a "worker" consumes jobs through a
// Consumer. Task-type strings and payload shapes are identical across all
// backends, so a payload enqueued by one transport is understood by another's
// handlers unchanged.
//
// All three Publisher implementations satisfy the small EmailQueue interface
// used by modules/user/service (EnqueueEmail), so any backend can back the
// welcome-email flow without touching the caller.
package queue

import "context"

// ── Task types ────────────────────────────────────────────────────────────────
// Kept identical to pkg/worker's constants so payloads are interchangeable
// across transports.

const (
	TaskSendEmail        = "email:send"
	TaskSendNotification = "notification:send"
	TaskBroadcast        = "notification:broadcast"
	TaskCleanupSessions  = "session:cleanup"
)

// ── Payload types ─────────────────────────────────────────────────────────────

// SendEmailPayload is a pre-rendered email handed to the worker for delivery.
// Matches pkg/worker.SendEmailPayload field-for-field (same JSON tags).
type SendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`           // HTML body
	Text    string `json:"text,omitempty"` // optional plain-text alternative
}

// ── Delivery ──────────────────────────────────────────────────────────────────

// Delivery is a backend-agnostic consumed message. Type is the task-type string
// (one of the constants above); Payload is the raw JSON body a handler unmarshals.
// It is the transport-neutral analogue of *asynq.Task.
type Delivery struct {
	Type    string
	Payload []byte
}

// ── Contract ──────────────────────────────────────────────────────────────────

// HandlerFunc processes a single delivered job. Returning a non-nil error tells
// the transport the job was not handled (nack / no-commit), so it may be retried
// or dead-lettered per that backend's semantics.
type HandlerFunc func(ctx context.Context, d Delivery) error

// Publisher enqueues jobs. Mirrors pkg/worker.Client (minus Asynq-specific
// options). Implementations must be safe for concurrent use.
type Publisher interface {
	// Enqueue marshals payload to JSON and publishes it under taskType.
	Enqueue(taskType string, payload any) error
	// EnqueueEmail is a convenience wrapper for the email:send task. It exists on
	// every backend so callers can depend on a tiny interface instead of the
	// concrete transport — see modules/user/service EmailQueue.
	EnqueueEmail(to, subject, html, text string) error
	// Close releases the underlying connection.
	Close() error
}

// Consumer runs a worker: register handlers with Handle, then Start to process
// jobs until the context is canceled. Mirrors pkg/worker.Server.
type Consumer interface {
	// Handle registers fn for a task type. Call before Start.
	Handle(taskType string, fn HandlerFunc)
	// Start begins consuming and blocks until ctx is canceled or a fatal error
	// occurs. Returns nil on a clean context-cancel shutdown.
	Start(ctx context.Context) error
	// Close releases the underlying connection.
	Close() error
}
