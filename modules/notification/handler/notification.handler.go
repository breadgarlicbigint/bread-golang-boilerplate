package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
	notifDTO  "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/dto"
	notifEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/entity"
	"github.com/google/uuid"
)

type NotifSvc interface {
	Send(ctx context.Context, req notifDTO.SendRequest) error
	Broadcast(ctx context.Context, req notifDTO.BroadcastRequest) (int, int, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int, unreadOnly bool) ([]*notifEntity.Notification, int64, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkRead(ctx context.Context, notifID, userID uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) error
	RegisterDevice(ctx context.Context, userID uuid.UUID, req notifDTO.RegisterDeviceRequest) error
	RemoveDevice(ctx context.Context, userID uuid.UUID, token string) error
	GetPreferences(ctx context.Context, userID uuid.UUID) (*notifEntity.NotificationPreferences, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, req notifDTO.UpdatePreferencesRequest) error
}

type NotificationHandler struct {
	svc NotifSvc
}

func New(svc NotifSvc) *NotificationHandler {
	return &NotificationHandler{svc: svc}
}

// RegisterRoutes mounts all notification endpoints.
func (h *NotificationHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc, adminMw gin.HandlerFunc) {
	me := rg.Group("/me/notifications", authMw)
	me.GET("",                  h.List)
	me.GET("/unread-count",     h.UnreadCount)
	me.PATCH("/:id/read",      h.MarkRead)
	me.PATCH("/read-all",      h.MarkAllRead)
	me.GET("/preferences",     h.GetPreferences)
	me.PATCH("/preferences",   h.UpdatePreferences)
	me.POST("/devices",        h.RegisterDevice)
	me.DELETE("/devices/:token", h.RemoveDevice)

	admin := rg.Group("/admin/notifications", authMw, adminMw)
	admin.POST("/send",      h.AdminSend)
	admin.POST("/broadcast", h.Broadcast)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := mustUserID(c)
	q := pagination.FromContext(c)
	unreadOnly := c.Query("unreadOnly") == "true"

	notifs, total, err := h.svc.ListByUser(c.Request.Context(), userID, q.Page, q.PerPage, unreadOnly)
	if err != nil {
		handleErr(c, err)
		return
	}
	meta := q.BuildMeta(total)
	resp := make([]notifDTO.NotificationResponse, len(notifs))
	for i, n := range notifs {
		resp[i] = toResponse(n)
	}
	response.OKWithMeta(c, "Notifications fetched", resp, &response.Meta{
		Total: meta.Total, Page: meta.Page, PerPage: meta.PerPage,
		TotalPage: meta.TotalPage, HasNext: meta.HasNext, HasPrev: meta.HasPrev,
	})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	count, err := h.svc.UnreadCount(c.Request.Context(), mustUserID(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Unread count fetched", notifDTO.UnreadCountResponse{Unread: count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid notification ID")
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), notifID, mustUserID(c)); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Marked as read", nil)
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	if err := h.svc.MarkAllRead(c.Request.Context(), mustUserID(c)); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "All notifications marked as read", nil)
}

func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	prefs, err := h.svc.GetPreferences(c.Request.Context(), mustUserID(c))
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Preferences fetched", prefs)
}

func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	var req notifDTO.UpdatePreferencesRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdatePreferences(c.Request.Context(), mustUserID(c), req); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Preferences updated", nil)
}

func (h *NotificationHandler) RegisterDevice(c *gin.Context) {
	var req notifDTO.RegisterDeviceRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.RegisterDevice(c.Request.Context(), mustUserID(c), req); err != nil {
		handleErr(c, err)
		return
	}
	response.Created(c, "Device registered", nil)
}

func (h *NotificationHandler) RemoveDevice(c *gin.Context) {
	if err := h.svc.RemoveDevice(c.Request.Context(), mustUserID(c), c.Param("token")); err != nil {
		handleErr(c, err)
		return
	}
	response.NoContent(c)
}

func (h *NotificationHandler) AdminSend(c *gin.Context) {
	var req notifDTO.SendRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.Send(c.Request.Context(), req); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Notification sent", nil)
}

func (h *NotificationHandler) Broadcast(c *gin.Context) {
	var req notifDTO.BroadcastRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	success, failed, err := h.svc.Broadcast(c.Request.Context(), req)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Broadcast complete", gin.H{"success": success, "failed": failed})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustUserID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	return id
}

func toResponse(n *notifEntity.Notification) notifDTO.NotificationResponse {
	return notifDTO.NotificationResponse{
		ID:        n.ID.String(),
		Type:      n.Type,
		Channel:   n.Channel,
		Status:    n.Status,
		Title:     n.Title,
		Body:      n.Body,
		ImageURL:  n.ImageURL,
		Data:      n.Data,
		ActionURL: n.ActionURL,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt.UTC().String(),
	}
}

func handleErr(c *gin.Context, err error) {
	if ae, ok := errors.As(err); ok {
		response.Error(c, ae.Status, ae.Message)
		return
	}
	response.LogInternal(err, "unexpected error")
	response.InternalServerError(c, "An unexpected error occurred")
}
