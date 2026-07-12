package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
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
	SendTestEmail(ctx context.Context, to string) error
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
	admin.POST("/send",       h.AdminSend)
	admin.POST("/broadcast",  h.Broadcast)
	admin.POST("/test-email", h.TestEmail)
}

func (h *NotificationHandler) List(c *gin.Context) {
	userID := mustUserID(c)
	q := pagination.FromContext(c)
	unreadOnly := c.Query("unreadOnly") == "true"

	notifs, total, err := h.svc.ListByUser(c.Request.Context(), userID, q.Page, q.PerPage, unreadOnly)
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	meta := q.BuildMeta(total)
	resp := make([]notifDTO.NotificationResponse, len(notifs))
	for i, n := range notifs {
		resp[i] = toResponse(n)
	}
	response.OKWithMetaI18n(c, "notification.listSuccess", resp, &response.Meta{
		Total: meta.Total, Page: meta.Page, PerPage: meta.PerPage,
		TotalPage: meta.TotalPage, HasNext: meta.HasNext, HasPrev: meta.HasPrev,
	})
}

func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	count, err := h.svc.UnreadCount(c.Request.Context(), mustUserID(c))
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.unreadCountFetched", notifDTO.UnreadCountResponse{Unread: count})
}

func (h *NotificationHandler) MarkRead(c *gin.Context) {
	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.ErrorI18n(c, http.StatusBadRequest, "notification.invalidID")
		return
	}
	if err := h.svc.MarkRead(c.Request.Context(), notifID, mustUserID(c)); err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.markReadSuccess", nil)
}

func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	if err := h.svc.MarkAllRead(c.Request.Context(), mustUserID(c)); err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.markAllReadSuccess", nil)
}

func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	prefs, err := h.svc.GetPreferences(c.Request.Context(), mustUserID(c))
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.prefsFetched", prefs)
}

func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	var req notifDTO.UpdatePreferencesRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.UpdatePreferences(c.Request.Context(), mustUserID(c), req); err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.prefsUpdated", nil)
}

func (h *NotificationHandler) RegisterDevice(c *gin.Context) {
	var req notifDTO.RegisterDeviceRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.RegisterDevice(c.Request.Context(), mustUserID(c), req); err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.CreatedI18n(c, "notification.deviceRegistered", nil)
}

func (h *NotificationHandler) RemoveDevice(c *gin.Context) {
	if err := h.svc.RemoveDevice(c.Request.Context(), mustUserID(c), c.Param("token")); err != nil {
		response.HandleAppError(c, err)
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
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.sendSuccess", nil)
}

func (h *NotificationHandler) Broadcast(c *gin.Context) {
	var req notifDTO.BroadcastRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	success, failed, err := h.svc.Broadcast(c.Request.Context(), req)
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "notification.broadcastComplete", gin.H{"success": success, "failed": failed})
}

// TestEmail godoc
// @Summary     Send a test email to verify MAIL_DRIVER configuration
// @Description Admin diagnostic endpoint — sends a minimal message through whichever mail transport is configured (SES or SMTP) and reports the raw result (including connection/auth errors), so email delivery can be debugged without creating a real user.
// @Tags        notifications
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body notifDTO.TestEmailRequest true "Recipient address"
// @Success     200 {object} notifDTO.TestEmailResponse
// @Failure     422 {object} response.ErrorEnvelope
// @Router      /v1/admin/notifications/test-email [post]
func (h *NotificationHandler) TestEmail(c *gin.Context) {
	var req notifDTO.TestEmailRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SendTestEmail(c.Request.Context(), req.To); err != nil {
		response.OKI18n(c, "notification.testEmailFailed", notifDTO.TestEmailResponse{Sent: false, Error: err.Error()})
		return
	}
	response.OKI18n(c, "notification.testEmailSent", notifDTO.TestEmailResponse{Sent: true})
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

