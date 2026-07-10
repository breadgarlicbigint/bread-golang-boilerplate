package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/mobile/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/sms"
	"github.com/google/uuid"
)

type MobileSvc interface {
	SendOTP(ctx context.Context, userID uuid.UUID, e164, correlationID string, channel sms.Channel) error
	VerifyOTP(ctx context.Context, userID uuid.UUID, e164, code string) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.UserMobile, error)
	SetPrimary(ctx context.Context, userID uuid.UUID, e164 string) error
	Delete(ctx context.Context, userID uuid.UUID, e164 string) error
}

type MobileHandler struct {
	svc MobileSvc
}

func New(svc MobileSvc) *MobileHandler {
	return &MobileHandler{svc: svc}
}

// RegisterRoutes mounts mobile verification endpoints under /v1/me/mobiles.
func (h *MobileHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc) {
	me := rg.Group("/me/mobiles", authMw)
	{
		me.GET("", h.List)
		me.POST("/send-otp", h.SendOTP)
		me.POST("/verify", h.Verify)
		me.PATCH("/:e164/primary", h.SetPrimary)
		me.DELETE("/:e164", h.Delete)
	}
}

// SendOTP godoc
// @Summary     Send OTP to a mobile number
// @Description Sends a 6-digit code via SMS or WhatsApp. channel: "sms" | "whatsapp"
// @Security    BearerAuth
// @Tags        mobile
// @Accept      json
// @Produce     json
// @Param       body body object{e164=string,channel=string} true "Phone number in E.164 format and channel"
// @Success     200
// @Router      /v1/me/mobiles/send-otp [post]
func (h *MobileHandler) SendOTP(c *gin.Context) {
	var req struct {
		E164    string `json:"e164"    binding:"required"`
		Channel string `json:"channel"` // "sms" | "whatsapp"  default: "sms"
	}
	if !validate.BindJSON(c, &req) {
		return
	}

	channel := sms.ChannelSMS
	if req.Channel == "whatsapp" {
		channel = sms.ChannelWhatsApp
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	correlationID := c.GetString("requestId")

	if err := h.svc.SendOTP(c.Request.Context(), userID, req.E164, correlationID, channel); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "OTP sent successfully", gin.H{"channel": string(channel), "e164": req.E164})
}

// Verify godoc
// @Summary     Verify mobile OTP
// @Security    BearerAuth
// @Tags        mobile
// @Accept      json
// @Produce     json
// @Param       body body object{e164=string,code=string} true "Phone number and 6-digit code"
// @Success     200
// @Router      /v1/me/mobiles/verify [post]
func (h *MobileHandler) Verify(c *gin.Context) {
	var req struct {
		E164 string `json:"e164" binding:"required"`
		Code string `json:"code" binding:"required,len=6"`
	}
	if !validate.BindJSON(c, &req) {
		return
	}

	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.svc.VerifyOTP(c.Request.Context(), userID, req.E164, req.Code); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Mobile number verified", nil)
}

// List godoc
// @Summary     List verified mobile numbers
// @Security    BearerAuth
// @Tags        mobile
// @Produce     json
// @Success     200 {array} entity.UserMobile
// @Router      /v1/me/mobiles [get]
func (h *MobileHandler) List(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	mobiles, err := h.svc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Mobiles fetched", mobiles)
}

// SetPrimary godoc
// @Summary     Set primary mobile number
// @Security    BearerAuth
// @Tags        mobile
// @Param       e164 path string true "E.164 phone number"
// @Success     200
// @Router      /v1/me/mobiles/{e164}/primary [patch]
func (h *MobileHandler) SetPrimary(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.svc.SetPrimary(c.Request.Context(), userID, c.Param("e164")); err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Primary mobile updated", nil)
}

// Delete godoc
// @Summary     Remove a mobile number
// @Security    BearerAuth
// @Tags        mobile
// @Param       e164 path string true "E.164 phone number"
// @Success     204
// @Router      /v1/me/mobiles/{e164} [delete]
func (h *MobileHandler) Delete(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString(middleware.CtxUserID))
	if err := h.svc.Delete(c.Request.Context(), userID, c.Param("e164")); err != nil {
		handleErr(c, err)
		return
	}
	response.NoContent(c)
}

func handleErr(c *gin.Context, err error) {
	if ae, ok := errors.As(err); ok {
		response.Error(c, ae.Status, ae.Message)
		return
	}
	response.LogInternal(err, "unexpected error")
	response.InternalServerError(c, "An unexpected error occurred")
}
