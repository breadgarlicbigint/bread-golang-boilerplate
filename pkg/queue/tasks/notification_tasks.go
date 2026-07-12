package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	notifDTO "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
	"go.uber.org/zap"
)

// NotificationSvc is the subset of the notification service the worker needs.
// Identical to pkg/worker/tasks.NotificationSvc.
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
func (h *NotificationTaskHandler) HandleSend(ctx context.Context, d queue.Delivery) error {
	var req notifDTO.SendRequest
	if err := json.Unmarshal(d.Payload, &req); err != nil {
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
func (h *NotificationTaskHandler) HandleBroadcast(ctx context.Context, d queue.Delivery) error {
	var req notifDTO.BroadcastRequest
	if err := json.Unmarshal(d.Payload, &req); err != nil {
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

// RegisterNotifications wires notification handlers onto any queue.Consumer.
func RegisterNotifications(c queue.Consumer, h *NotificationTaskHandler) {
	c.Handle(queue.TaskSendNotification, h.HandleSend)
	c.Handle(queue.TaskBroadcast, h.HandleBroadcast)
}
