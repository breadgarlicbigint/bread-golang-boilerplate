package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

type RoleSvc interface {
	List(ctx context.Context) ([]entity.Role, error)
}

type RoleHandler struct {
	svc RoleSvc
}

func New(svc RoleSvc) *RoleHandler {
	return &RoleHandler{svc: svc}
}

// RegisterRoutes mounts role endpoints. extraMw is typically authMw + adminMw.
func (h *RoleHandler) RegisterRoutes(rg *gin.RouterGroup, extraMw ...gin.HandlerFunc) {
	roles := rg.Group("/roles", extraMw...)
	roles.GET("", h.List)
}

// List godoc
// @Summary     List all roles
// @Description Used to populate role-selection dropdowns (e.g. admin "create user" form)
// @Tags        roles
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} dto.RoleResponse
// @Failure     401 {object} response.ErrorEnvelope
// @Failure     403 {object} response.ErrorEnvelope
// @Router      /v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.svc.List(c.Request.Context())
	if err != nil {
		response.HandleAppError(c, err)
		return
	}

	result := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		result = append(result, dto.FromEntity(r))
	}
	response.OKI18n(c, "role.listSuccess", result)
}
