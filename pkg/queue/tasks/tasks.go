// Package tasks holds transport-agnostic job handlers for the RabbitMQ and Kafka
// workers. Each handler takes a queue.Delivery (not a backend-specific message),
// so the same handler set is registered by apps/worker-rabbitmq and
// apps/worker-kafka unchanged. It is the parallel of pkg/worker/tasks, which
// serves the Redis/Asynq worker with *asynq.Task handlers.
package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
	"go.uber.org/zap"
)

// EmailTaskHandler delivers pre-rendered emails for the email:send task.
type EmailTaskHandler struct {
	mailer *email.Mailer
	log    *zap.Logger
}

func NewEmailTaskHandler(mailer *email.Mailer, log *zap.Logger) *EmailTaskHandler {
	return &EmailTaskHandler{mailer: mailer, log: log}
}

func (h *EmailTaskHandler) Handle(ctx context.Context, d queue.Delivery) error {
	var p queue.SendEmailPayload
	if err := json.Unmarshal(d.Payload, &p); err != nil {
		return fmt.Errorf("tasks: unmarshal email payload: %w", err)
	}

	h.log.Info("processing email task",
		zap.String("taskType", d.Type), zap.String("to", p.To), zap.String("subject", p.Subject))

	if h.mailer == nil {
		h.log.Warn("mailer not configured — skipping email task")
		return nil
	}

	if err := h.mailer.Send(ctx, email.Message{
		To:      []string{p.To},
		Subject: p.Subject,
		HTML:    p.Body,
		Text:    p.Text,
	}); err != nil {
		h.log.Error("email send failed", zap.String("to", p.To), zap.Error(err))
		return err
	}
	return nil
}

// SessionCleanupHandler is a hook for periodic session housekeeping. Redis TTLs
// handle expiry automatically; this mirrors the Asynq worker's equivalent task.
type SessionCleanupHandler struct {
	log *zap.Logger
}

func NewSessionCleanupHandler(log *zap.Logger) *SessionCleanupHandler {
	return &SessionCleanupHandler{log: log}
}

func (h *SessionCleanupHandler) Handle(_ context.Context, _ queue.Delivery) error {
	h.log.Info("running session cleanup task")
	return nil
}

// RegisterAll wires the core handlers onto any queue.Consumer (RabbitMQ or
// Kafka). Transactional and promotional email share one handler — routing
// between backends (see pkg/queue/router) happens on the enqueue side, not here.
func RegisterAll(c queue.Consumer, emailH *EmailTaskHandler, sessionH *SessionCleanupHandler) {
	c.Handle(queue.TaskSendEmail, emailH.Handle)
	c.Handle(queue.TaskSendPromotionalEmail, emailH.Handle)
	c.Handle(queue.TaskCleanupSessions, sessionH.Handle)
}
