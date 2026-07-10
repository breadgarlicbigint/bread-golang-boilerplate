package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
)

type AuthSvc interface {
	Login(ctx context.Context, req dto.LoginRequest, ip string) (*dto.LoginResponse, error)
	Register(ctx context.Context, req dto.RegisterRequest, ip string) (*dto.LoginResponse, error)
	Refresh(ctx context.Context, refreshToken string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, sessionID string) error
	LogoutAll(ctx context.Context, userID string) error
	Enable2FA(ctx context.Context, userID, email string) (*dto.Enable2FAResponse, error)
	Verify2FA(ctx context.Context, userID, code, secret string) error
}

type AuthHandler struct {
	svc AuthSvc
}

func New(svc AuthSvc) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc) {
	pub := rg.Group("/auth")
	pub.POST("/login",    h.Login)
	pub.POST("/register", h.Register)
	pub.POST("/refresh",  h.Refresh)

	protected := rg.Group("/auth", authMw)
	protected.DELETE("/logout",     h.Logout)
	protected.DELETE("/logout-all", h.LogoutAll)
	protected.POST("/2fa/enable",   h.Enable2FA)
	protected.POST("/2fa/verify",   h.Verify2FA)
}

// Login godoc
// @Summary     Login with email & password
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.LoginRequest true "Credentials"
// @Success     200 {object} dto.LoginResponse
// @Failure     400 {object} response.ErrorEnvelope
// @Failure     401 {object} response.ErrorEnvelope
// @Failure     422 {object} response.ErrorEnvelope
// @Router      /v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Login successful", resp)
}

// Register godoc
// @Summary     Register a new account
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.RegisterRequest true "Registration payload"
// @Success     201 {object} dto.LoginResponse
// @Failure     400 {object} response.ErrorEnvelope
// @Failure     409 {object} response.ErrorEnvelope
// @Failure     422 {object} response.ErrorEnvelope
// @Router      /v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Register(c.Request.Context(), req, c.ClientIP())
	if err != nil {
		handleError(c, err)
		return
	}
	response.Created(c, "Registration successful", resp)
}

// Refresh godoc
// @Summary     Refresh access token
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body dto.RefreshRequest true "Refresh token"
// @Success     200 {object} dto.LoginResponse
// @Failure     400 {object} response.ErrorEnvelope
// @Failure     401 {object} response.ErrorEnvelope
// @Router      /v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	resp, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Token refreshed", resp)
}

// Logout godoc
// @Summary     Logout (revoke current session)
// @Security    BearerAuth
// @Tags        auth
// @Success     200
// @Failure     401 {object} response.ErrorEnvelope
// @Router      /v1/auth/logout [delete]
func (h *AuthHandler) Logout(c *gin.Context) {
	if err := h.svc.Logout(c.Request.Context(), c.GetString(middleware.CtxSessionID)); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Logged out successfully", nil)
}

// LogoutAll godoc
// @Summary     Logout from all devices
// @Security    BearerAuth
// @Tags        auth
// @Success     200
// @Router      /v1/auth/logout-all [delete]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	if err := h.svc.LogoutAll(c.Request.Context(), c.GetString(middleware.CtxUserID)); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Logged out from all devices", nil)
}

// Enable2FA godoc
// @Summary     Enable two-factor authentication
// @Security    BearerAuth
// @Tags        auth
// @Success     200 {object} dto.Enable2FAResponse
// @Router      /v1/auth/2fa/enable [post]
func (h *AuthHandler) Enable2FA(c *gin.Context) {
	resp, err := h.svc.Enable2FA(c.Request.Context(), c.GetString(middleware.CtxUserID), "")
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "2FA setup initiated. Scan the QR code with your authenticator app.", resp)
}

// Verify2FA godoc
// @Summary     Verify and activate 2FA
// @Security    BearerAuth
// @Tags        auth
// @Accept      json
// @Param       body body dto.Verify2FARequest true "TOTP code"
// @Success     200
// @Failure     400 {object} response.ErrorEnvelope
// @Failure     422 {object} response.ErrorEnvelope
// @Router      /v1/auth/2fa/verify [post]
func (h *AuthHandler) Verify2FA(c *gin.Context) {
	var req dto.Verify2FARequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.Verify2FA(c.Request.Context(), c.GetString(middleware.CtxUserID), req.Code, ""); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Two-factor authentication activated", nil)
}

func handleError(c *gin.Context, err error) {
	if ae, ok := errors.As(err); ok {
		response.Error(c, ae.Status, ae.Message)
		return
	}
	response.LogInternal(err, "unexpected error")
	response.InternalServerError(c, "An unexpected error occurred")
}
