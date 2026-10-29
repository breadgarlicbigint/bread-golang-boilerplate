package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
)

type AppVersionSvc interface {
	Check(ctx context.Context, platform service.Platform, clientVersion string) (*service.VersionCheckResponse, error)
	Upsert(ctx context.Context, platform service.Platform, current, min string, force bool, notes, storeURL string) (*service.AppVersion, error)
	List(ctx context.Context) ([]service.AppVersion, error)
	Release(ctx context.Context, in service.ReleaseInput) (*service.AppVersion, error)
	DownloadURL(ctx context.Context, platform service.Platform, requestBase string) (string, error)
}

type AppVersionHandler struct {
	svc AppVersionSvc
	// maxBytes caps the multipart release artefact (0 ⇒ defaultMaxArtifactBytes).
	maxBytes int64
}

func New(svc AppVersionSvc) *AppVersionHandler {
	return &AppVersionHandler{svc: svc}
}

// RegisterRoutes mounts version endpoints.
// authMw and adminMw are variadic so the caller can pass 0-2 middleware.
func (h *AppVersionHandler) RegisterRoutes(rg *gin.RouterGroup, extraMw ...gin.HandlerFunc) {
	// Public — mobile apps call on launch
	rg.GET("/app-version/check", h.Check)

	// Public — stable download link (safe in an email; resolves a fresh URL)
	rg.GET("/app-version/download", h.Download)

	// Admin — manage policies
	admin := rg.Group("/admin/app-versions", extraMw...)
	admin.GET("", h.List)
	admin.PUT("/:platform", h.Upsert)
}

// RegisterReleaseRoute mounts the CI publish endpoint on its own gate, SEPARATE
// from RegisterRoutes: it is the only route whose caller is a machine rather
// than a person — it takes an API key and no session, belongs to no tenant, and
// is the one route that can put every device behind a mandatory update.
//
// apiKeyMw is required, not variadic — a nil gate here is an anonymous
// app-bricking lever the caller must be unable to forget. The route carries NO
// swagger annotations deliberately, so it is absent from the published spec;
// swagger_absence_test.go asserts that so the omission can't be undone.
func (h *AppVersionHandler) RegisterReleaseRoute(rg *gin.RouterGroup, apiKeyMw gin.HandlerFunc, maxBytes int64) {
	h.maxBytes = maxBytes
	rg.POST("/app-version/release", apiKeyMw, h.Release)
}

// Download redirects to the current release's artefact (or its store page). A
// stable link: the presigned URL it resolves to is minted per request.
//
// @Summary     Download the latest app build
// @Tags        versioning
// @Param       platform query string false "android (default) | ios"
// @Success     302
// @Failure     404 {object} response.ErrorEnvelope
// @Router      /v1/app-version/download [get]
func (h *AppVersionHandler) Download(c *gin.Context) {
	platform := service.Platform(c.DefaultQuery("platform", string(service.PlatformAndroid)))
	switch platform {
	case service.PlatformAndroid, service.PlatformIOS:
	default:
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.invalidPlatform")
		return
	}
	url, err := h.svc.DownloadURL(c.Request.Context(), platform, middleware.RequestBase(c))
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	if url == "" {
		response.ErrorI18n(c, http.StatusNotFound, "appVersion.noDownload")
		return
	}
	// 302, not 301: the target changes with every release and must never be
	// cached by a browser as the link's permanent answer.
	c.Redirect(http.StatusFound, url)
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
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.missingParams")
		return
	}

	result, err := h.svc.Check(c.Request.Context(), service.Platform(platformStr), version)
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "appVersion.checkSuccess", result)
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
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "appVersion.listSuccess", versions)
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
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.invalidPlatform")
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
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "appVersion.upsertSuccess", av)
}
