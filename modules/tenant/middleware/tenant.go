package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/entity"
)

// Context keys set by the tenant middleware.
const (
	CtxTenantID   = "tenantId"
	CtxTenantSlug = "tenantSlug"
	CtxTenantObj  = "tenant"
)

// TenantResolver finds a tenant by slug or domain.
type TenantResolver interface {
	FindBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
	FindByDomain(ctx context.Context, domain string) (*entity.Tenant, error)
}

// TenantFromHeader resolves the tenant from X-Tenant-ID (slug) header.
func TenantFromHeader(resolver TenantResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.GetHeader("X-Tenant-ID")
		if slug == "" {
			response.ErrorI18n(c, http.StatusBadRequest, "tenant.headerRequired")
			return
		}
		resolveTenant(c, resolver, slug)
	}
}

// TenantFromSubdomain resolves the tenant from the request subdomain.
func TenantFromSubdomain(resolver TenantResolver, baseDomain string) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		if !strings.HasSuffix(host, "."+baseDomain) {
			t, err := resolver.FindByDomain(c.Request.Context(), host)
			if err == nil {
				setTenantContext(c, t)
				c.Next()
				return
			}
		}
		slug := strings.TrimSuffix(host, "."+baseDomain)
		if slug == "" || slug == host {
			response.ErrorI18n(c, http.StatusBadRequest, "tenant.hostUnresolved")
			return
		}
		resolveTenant(c, resolver, slug)
	}
}

// TenantFromQuery resolves from ?tenant=<slug> (useful for development).
func TenantFromQuery(resolver TenantResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Query("tenant")
		if slug == "" {
			response.ErrorI18n(c, http.StatusBadRequest, "tenant.queryParamRequired")
			return
		}
		resolveTenant(c, resolver, slug)
	}
}

// TenantOptional tries X-Tenant-ID but does not abort if missing.
func TenantOptional(resolver TenantResolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.GetHeader("X-Tenant-ID")
		if slug != "" {
			resolveTenant(c, resolver, slug)
			return
		}
		c.Next()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func resolveTenant(c *gin.Context, r TenantResolver, slug string) {
	t, err := r.FindBySlug(c.Request.Context(), slug)
	if err != nil {
		response.NotFoundI18n(c, "tenant.notFound")
		return
	}
	if t.Status == entity.TenantStatusSuspended {
		response.ForbiddenI18n(c, "tenant.suspended")
		return
	}
	setTenantContext(c, t)
	c.Next()
}

func setTenantContext(c *gin.Context, t *entity.Tenant) {
	c.Set(CtxTenantID, t.ID.String())
	c.Set(CtxTenantSlug, t.Slug)
	c.Set(CtxTenantObj, t)
}
