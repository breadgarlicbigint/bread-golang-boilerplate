package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/worker"
	"go.uber.org/zap"
)

// EmailTaskHandler handles the email:send task.
type EmailTaskHandler struct {
	mailer *email.Mailer
	log    *zap.Logger
}

func NewEmailTaskHandler(mailer *email.Mailer, log *zap.Logger) *EmailTaskHandler {
	return &EmailTaskHandler{mailer: mailer, log: log}
}

func (h *EmailTaskHandler) Handle(ctx context.Context, t *asynq.Task) error {
	var p worker.SendEmailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("tasks: unmarshal email payload: %w", err)
	}

	h.log.Info("processing email task",
		zap.String("taskType", t.Type()), zap.String("to", p.To), zap.String("subject", p.Subject))

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

// SessionCleanupHandler purges expired session keys from Redis.
type SessionCleanupHandler struct {
	log *zap.Logger
}

func NewSessionCleanupHandler(log *zap.Logger) *SessionCleanupHandler {
	return &SessionCleanupHandler{log: log}
}

func (h *SessionCleanupHandler) Handle(ctx context.Context, t *asynq.Task) error {
	h.log.Info("running session cleanup task")
	// Redis TTL handles expiry automatically; this task is a hook for custom logic.
	return nil
}

// RegisterAll registers all task handlers on the worker server. Transactional
// and promotional email share one handler — routing between backends (see
// pkg/queue/router) happens on the enqueue side, not here.
func RegisterAll(srv *worker.Server, emailH *EmailTaskHandler, sessionH *SessionCleanupHandler) {
	srv.Handle(worker.TaskSendEmail, emailH.Handle)
	srv.Handle(worker.TaskSendPromotionalEmail, emailH.Handle)
	srv.Handle(worker.TaskCleanupExpiredSessions, sessionH.Handle)
}
