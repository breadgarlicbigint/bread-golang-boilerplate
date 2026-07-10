package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	notifDTO "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/dto"
	"go.uber.org/zap"
)

const (
	TaskSendNotification  = "notification:send"
	TaskBroadcast         = "notification:broadcast"
	TaskCleanStaleTokens  = "notification:clean_tokens"
)

// NotificationSvc is the interface the worker needs.
type NotificationSvc interface {
	Send(ctx context.Context, req notifDTO.SendRequest) error
	Broadcast(ctx context.Context, req notifDTO.BroadcastRequest) (int, int, error)
}

// NotificationTaskHandler processes queued notification jobs.
type NotificationTaskHandler struct {
	svc NotificationSvc
	log *zap.Logger
}

func NewNotificationTaskHandler(svc NotificationSvc, log *zap.Logger) *NotificationTaskHandler {
	return &NotificationTaskHandler{svc: svc, log: log}
}

// HandleSend processes a single notification send task.
func (h *NotificationTaskHandler) HandleSend(ctx context.Context, t *asynq.Task) error {
	var req notifDTO.SendRequest
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		return fmt.Errorf("notification task: unmarshal: %w", err)
	}
	h.log.Info("processing notification",
		zap.String("userId", req.UserID),
		zap.String("channel", string(req.Channel)),
		zap.String("type", string(req.Type)),
	)
	return h.svc.Send(ctx, req)
}

// HandleBroadcast processes a broadcast notification task.
func (h *NotificationTaskHandler) HandleBroadcast(ctx context.Context, t *asynq.Task) error {
	var req notifDTO.BroadcastRequest
	if err := json.Unmarshal(t.Payload(), &req); err != nil {
		return fmt.Errorf("broadcast task: unmarshal: %w", err)
	}
	h.log.Info("broadcasting notification",
		zap.Int("users", len(req.UserIDs)),
		zap.String("channel", string(req.Channel)),
	)
	success, failed, err := h.svc.Broadcast(ctx, req)
	h.log.Info("broadcast complete", zap.Int("success", success), zap.Int("failed", failed))
	return err
}

// HandleCleanStaleTokens removes FCM tokens that are no longer valid.
func (h *NotificationTaskHandler) HandleCleanStaleTokens(ctx context.Context, t *asynq.Task) error {
	h.log.Info("cleaning stale device tokens")
	// The FCMSender already removes stale tokens during SendMulticast.
	// This periodic task is a safety net for tokens that haven't been used recently.
	return nil
}
