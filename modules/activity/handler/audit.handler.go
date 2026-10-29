// Package handler exposes the audit trail for reading. There is deliberately no
// write, update or delete route: the trail is append-only, written only by the
// services that make the changes. An editable audit log is not an audit log.
package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	activitySvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/activity/service"
	roleentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/audit"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/export"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

// auditExportColumns is the ?format=xlsx layout for the trail. Headers resolve
// through the same fields. locale dictionary validation messages use.
var auditExportColumns = []export.Column{
	{Field: "at", Kind: export.Date},
	{Field: "action", Kind: export.Text},
	{Field: "entityType", Kind: export.Text},
	{Field: "entityId", Kind: export.Text},
	{Field: "actorId", Kind: export.Text},
	{Field: "actorRole", Kind: export.Text},
	{Field: "risk", Kind: export.Text},
}

// Svc is the handler's view of the audit service.
type Svc interface {
	List(ctx context.Context, tenantID uuid.UUID, f activitySvc.AuditFilter, q pagination.Query) ([]audit.Entry, int64, error)
	ListGrouped(ctx context.Context, tenantID uuid.UUID, f activitySvc.AuditFilter, q pagination.Query) ([]pagination.GroupedRow, int64, []pagination.GroupCount, error)
	ListGroupPaged(ctx context.Context, tenantID uuid.UUID, f activitySvc.AuditFilter, q pagination.Query) ([]any, int64, int64, error)
	EntityHistory(ctx context.Context, tenantID uuid.UUID, entityType, entityID string) ([]audit.Entry, error)
}

type AuditHandler struct{ svc Svc }

func New(svc Svc) *AuditHandler { return &AuditHandler{svc: svc} }

// RegisterRoutes mounts the read-only trail behind audit:read + RequireTenant.
func (h *AuditHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc) {
	read := middleware.RequirePermission(string(roleentity.PermAuditRead))
	g := rg.Group("/audit", authMw, middleware.RequireTenant())
	g.GET("", read, h.List)
	g.GET("/entity/:type/:id", read, h.EntityHistory)
}

func tenant(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(middleware.TenantIDFromCtx(c))
	return id
}

// List godoc
// @Summary      Read the audit trail (paginated, newest first)
// @Tags         audit
// @Security     BearerAuth
// @Produce      json
// @Param        actorId    query string false "Filter by actor"
// @Param        entityType query string false "Entity type"
// @Param        entityId   query string false "Filter to one record"
// @Param        action     query string false "Action"
// @Param        risk       query string false "NORMAL | HIGH"
// @Success      200 {array} audit.Entry
// @Router       /v1/audit [get]
func (h *AuditHandler) List(c *gin.Context) {
	f := activitySvc.AuditFilter{
		ActorID:    c.Query("actorId"),
		EntityType: c.Query("entityType"),
		EntityID:   c.Query("entityId"),
		Action:     c.Query("action"),
		Risk:       c.Query("risk"),
	}
	q := pagination.FromContext(c)

	// The whole filtered result set as a spreadsheet (?format=xlsx). Checked
	// BEFORE the grouping branches: grouping arranges rows on a screen, an export
	// carries the records themselves and inherits this route's gate/filters.
	if export.Requested(c) {
		rows, total, err := h.svc.List(c.Request.Context(), tenant(c), f, export.Query(q))
		if err != nil {
			response.HandleAppError(c, err)
			return
		}
		export.Respond(c, "audit", auditExportColumns, rows, total)
		return
	}

	// Group-paged: the page windows the VISIBLE sequence (headers + rows of open
	// groups). Checked before IsGrouped, which it implies.
	if q.IsGroupPaged() {
		items, total, visible, err := h.svc.ListGroupPaged(c.Request.Context(), tenant(c), f, q)
		if err != nil {
			response.HandleAppError(c, err)
			return
		}
		m := q.BuildMetaVisible(total, visible)
		response.OKWithMetaI18n(c, "audit.listSuccess", items, &response.Meta{
			Total: m.Total, VisibleTotal: m.VisibleTotal, Page: m.Page, PerPage: m.PerPage,
			TotalPage: m.TotalPage, HasNext: m.HasNext, HasPrev: m.HasPrev,
		})
		return
	}

	// Grouped: the same page of RECORDS, wrapped with each entry's group path.
	// Paging and `total` are unchanged from ungrouped.
	if q.IsGrouped() {
		rows, total, groups, err := h.svc.ListGrouped(c.Request.Context(), tenant(c), f, q)
		if err != nil {
			response.HandleAppError(c, err)
			return
		}
		m := q.BuildMeta(total)
		response.OKWithMetaI18n(c, "audit.listSuccess", rows, &response.Meta{
			Total: m.Total, Page: m.Page, PerPage: m.PerPage, TotalPage: m.TotalPage,
			HasNext: m.HasNext, HasPrev: m.HasPrev, Groups: groups,
		})
		return
	}

	rows, total, err := h.svc.List(c.Request.Context(), tenant(c), f, q)
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	m := q.BuildMeta(total)
	response.OKWithMetaI18n(c, "audit.listSuccess", rows, &response.Meta{
		Total: m.Total, Page: m.Page, PerPage: m.PerPage, TotalPage: m.TotalPage, HasNext: m.HasNext, HasPrev: m.HasPrev,
	})
}

// EntityHistory godoc
// @Summary      Full change history for one record, oldest first
// @Tags         audit
// @Security     BearerAuth
// @Produce      json
// @Param        type path string true "Entity type"
// @Param        id   path string true "Entity id"
// @Success      200 {array} audit.Entry
// @Router       /v1/audit/entity/{type}/{id} [get]
func (h *AuditHandler) EntityHistory(c *gin.Context) {
	rows, err := h.svc.EntityHistory(c.Request.Context(), tenant(c), c.Param("type"), c.Param("id"))
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "audit.listSuccess", rows)
}
