package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
)

type AppVersionSvc interface {
	Check(ctx context.Context, platform service.Platform, clientVersion string) (*service.VersionCheckResponse, error)
	Upsert(ctx context.Context, platform service.Platform, current, min string, force bool, notes, storeURL string) (*service.AppVersion, error)
	List(ctx context.Context) ([]service.AppVersion, error)
}

type AppVersionHandler struct {
	svc AppVersionSvc
}

func New(svc AppVersionSvc) *AppVersionHandler {
	return &AppVersionHandler{svc: svc}
}

// RegisterRoutes mounts version endpoints.
// authMw and adminMw are variadic so the caller can pass 0-2 middleware.
func (h *AppVersionHandler) RegisterRoutes(rg *gin.RouterGroup, extraMw ...gin.HandlerFunc) {
	// Public — mobile apps call on launch
	rg.GET("/app-version/check", h.Check)

	// Admin — manage policies
	admin := rg.Group("/admin/app-versions", extraMw...)
	admin.GET("", h.List)
	admin.PUT("/:platform", h.Upsert)
}

// Check godoc
// @Summary     Check if app version is current
// @Tags        versioning
// @Produce     json
// @Param       X-App-Version  header string false "Client version e.g. 2.4.1"
// @Param       X-App-Platform header string false "ios | android | web"
// @Success     200 {object} service.VersionCheckResponse
// @Router      /v1/app-version/check [get]
func (h *AppVersionHandler) Check(c *gin.Context) {
	platformStr := c.GetHeader(middleware.HeaderPlatform)
	version := c.GetHeader(middleware.HeaderAppVersion)
	if platformStr == "" {
		platformStr = c.Query("platform")
	}
	if version == "" {
		version = c.Query("version")
	}
	if platformStr == "" || version == "" {
		response.BadRequest(c, "platform and version are required (header or query param)")
		return
	}

	result, err := h.svc.Check(c.Request.Context(), service.Platform(platformStr), version)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Version check complete", result)
}

// List godoc
// @Summary     List all platform version policies
// @Security    BearerAuth
// @Tags        versioning
// @Produce     json
// @Success     200 {array} service.AppVersion
// @Router      /v1/admin/app-versions [get]
func (h *AppVersionHandler) List(c *gin.Context) {
	versions, err := h.svc.List(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Versions fetched", versions)
}

// Upsert godoc
// @Summary     Create or update version policy
// @Security    BearerAuth
// @Tags        versioning
// @Accept      json
// @Param       platform path string true "ios | android | web"
// @Success     200 {object} service.AppVersion
// @Router      /v1/admin/app-versions/{platform} [put]
func (h *AppVersionHandler) Upsert(c *gin.Context) {
	platform := service.Platform(c.Param("platform"))
	switch platform {
	case service.PlatformIOS, service.PlatformAndroid, service.PlatformWeb:
	default:
		response.BadRequest(c, "Invalid platform. Use: ios | android | web")
		return
	}

	var req struct {
		CurrentVersion string `json:"currentVersion" binding:"required"`
		MinVersion     string `json:"minVersion"     binding:"required"`
		ForceUpdate    bool   `json:"forceUpdate"`
		ReleaseNotes   string `json:"releaseNotes"`
		StoreURL       string `json:"storeUrl"`
	}
	if !validate.BindJSON(c, &req) {
		return
	}

	av, err := h.svc.Upsert(c.Request.Context(), platform,
		req.CurrentVersion, req.MinVersion, req.ForceUpdate, req.ReleaseNotes, req.StoreURL)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Version policy updated", av)
}

func handleErr(c *gin.Context, err error) {
	if ae, ok := errors.As(err); ok {
		response.Error(c, ae.Status, ae.Message)
		return
	}
	response.LogInternal(err, "unexpected error")
	response.InternalServerError(c, "An unexpected error occurred")
}
