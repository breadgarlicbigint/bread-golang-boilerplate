package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	usersvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/service"
)

type UserSvc interface {
	Create(ctx context.Context, req dto.CreateUserRequest) (*entity.User, error)
	GetByID(ctx context.Context, id string) (*entity.User, error)
	List(ctx context.Context, q pagination.Query) ([]*entity.User, int64, error)
	Update(ctx context.Context, id string, req dto.UpdateUserRequest) (*entity.User, error)
	ChangePassword(ctx context.Context, id string, req dto.ChangePasswordRequest) error
	BlockUser(ctx context.Context, id, reason string) error
	UnblockUser(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

type UserHandler struct{ svc UserSvc }

func New(svc UserSvc) *UserHandler { return &UserHandler{svc: svc} }

func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc, adminMw gin.HandlerFunc) {
	// Admin routes — require auth + admin role
	admin := rg.Group("/users", authMw, adminMw)
	admin.POST("",           h.Create)
	admin.GET("",            h.List)
	admin.GET("/:id",        h.GetByID)
	admin.PATCH("/:id",      h.Update)
	admin.DELETE("/:id",     h.Delete)
	admin.POST("/:id/block", h.Block)
	admin.POST("/:id/unblock", h.Unblock)

	// Me routes — require auth only
	me := rg.Group("/me", authMw)
	me.GET("",             h.GetMe)
	me.PATCH("",           h.UpdateMe)
	me.PATCH("/password",  h.ChangePassword)
}

// Create godoc
// @Summary      Create user (admin)
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body dto.CreateUserRequest true "User payload"
// @Success      201 {object} dto.UserResponse
// @Failure      400 {object} response.ErrorEnvelope
// @Failure      409 {object} response.ErrorEnvelope
// @Failure      422 {object} response.ErrorEnvelope
// @Router       /v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req dto.CreateUserRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	u, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.Created(c, "User created", usersvc.MapToResponse(u))
}

// List godoc
// @Summary      List users (admin)
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Param        page    query int    false "Page number"
// @Param        perPage query int    false "Items per page"
// @Param        search  query string false "Search term"
// @Param        sortBy  query string false "Sort field"
// @Param        sortDir query string false "asc | desc"
// @Success      200 {object} dto.UserListResponse
// @Router       /v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	q := pagination.FromContext(c)
	users, total, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		handleError(c, err)
		return
	}
	meta := q.BuildMeta(total)
	resp := make([]dto.UserResponse, len(users))
	for i, u := range users {
		resp[i] = usersvc.MapToResponse(u)
	}
	response.OKWithMeta(c, "Users fetched", resp, &response.Meta{
		Total:     meta.Total,
		Page:      meta.Page,
		PerPage:   meta.PerPage,
		TotalPage: meta.TotalPage,
		HasNext:   meta.HasNext,
		HasPrev:   meta.HasPrev,
	})
}

// GetByID godoc
// @Summary      Get user by ID (admin)
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} dto.UserResponse
// @Failure      404 {object} response.ErrorEnvelope
// @Router       /v1/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	u, err := h.svc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "User fetched", usersvc.MapToResponse(u))
}

// GetMe godoc
// @Summary      Get my profile
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} dto.UserResponse
// @Router       /v1/me [get]
func (h *UserHandler) GetMe(c *gin.Context) {
	u, err := h.svc.GetByID(c.Request.Context(), c.GetString(middleware.CtxUserID))
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Profile fetched", usersvc.MapToResponse(u))
}

// Update godoc
// @Summary      Update user (admin)
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id   path string true "User ID"
// @Param        body body dto.UpdateUserRequest true "Update payload"
// @Success      200 {object} dto.UserResponse
// @Failure      400 {object} response.ErrorEnvelope
// @Failure      422 {object} response.ErrorEnvelope
// @Router       /v1/users/{id} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	var req dto.UpdateUserRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	u, err := h.svc.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "User updated", usersvc.MapToResponse(u))
}

// UpdateMe godoc
// @Summary      Update my profile
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body dto.UpdateUserRequest true "Update payload"
// @Success      200 {object} dto.UserResponse
// @Failure      422 {object} response.ErrorEnvelope
// @Router       /v1/me [patch]
func (h *UserHandler) UpdateMe(c *gin.Context) {
	var req dto.UpdateUserRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	u, err := h.svc.Update(c.Request.Context(), c.GetString(middleware.CtxUserID), req)
	if err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Profile updated", usersvc.MapToResponse(u))
}

// ChangePassword godoc
// @Summary      Change my password
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body dto.ChangePasswordRequest true "Password payload"
// @Success      200
// @Failure      400 {object} response.ErrorEnvelope
// @Failure      422 {object} response.ErrorEnvelope
// @Router       /v1/me/password [patch]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), c.GetString(middleware.CtxUserID), req); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "Password changed successfully", nil)
}

// Block godoc
// @Summary      Block a user (admin)
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Param        id   path string true "User ID"
// @Param        body body dto.BlockUserRequest true "Block reason"
// @Success      200
// @Failure      422 {object} response.ErrorEnvelope
// @Router       /v1/users/{id}/block [post]
func (h *UserHandler) Block(c *gin.Context) {
	var req dto.BlockUserRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.BlockUser(c.Request.Context(), c.Param("id"), req.Reason); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "User blocked", nil)
}

// Unblock godoc
// @Summary      Unblock a user (admin)
// @Tags         users
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      200
// @Router       /v1/users/{id}/unblock [post]
func (h *UserHandler) Unblock(c *gin.Context) {
	if err := h.svc.UnblockUser(c.Request.Context(), c.Param("id")); err != nil {
		handleError(c, err)
		return
	}
	response.OK(c, "User unblocked", nil)
}

// Delete godoc
// @Summary      Delete user (admin)
// @Tags         users
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Success      204
// @Router       /v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		handleError(c, err)
		return
	}
	response.NoContent(c)
}

func handleError(c *gin.Context, err error) {
	if ae, ok := errors.As(err); ok {
		response.Error(c, ae.Status, ae.Message)
		return
	}
	response.LogInternal(err, "unexpected error")
	response.InternalServerError(c, "An unexpected error occurred")
}
