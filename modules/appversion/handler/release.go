package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
)

// defaultMaxArtifactBytes is the fallback cap when the caller wires none. An APK
// with two ABIs runs 40-80 MB; 200 MB leaves room without accepting an upload
// that is obviously not a build.
const defaultMaxArtifactBytes int64 = 200 << 20

// Release publishes one build.
//
// NO SWAGGER ANNOTATIONS — see RegisterReleaseRoute. Do not add them;
// swagger_absence_test.go fails the build if you do.
//
// Multipart rather than JSON because the artefact and the policy that points at
// it must arrive together: a two-call shape can leave a policy advertising a
// build that was never uploaded — a mandatory update the whole estate cannot
// install.
func (h *AppVersionHandler) Release(c *gin.Context) {
	max := h.maxBytes
	if max <= 0 {
		max = defaultMaxArtifactBytes
	}
	// Capped before parsing: MultipartForm buffers to disk, so a check on the
	// parsed size has already paid for the whole upload.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			response.ErrorI18n(c, http.StatusRequestEntityTooLarge, "appVersion.artifactTooLarge")
			return
		case errors.Is(err, http.ErrNotMultipart):
			// A policy-only publish may arrive as a plain urlencoded form.
			if perr := c.Request.ParseForm(); perr != nil {
				response.ErrorI18n(c, http.StatusBadRequest, "validation.invalidBody")
				return
			}
		default:
			response.ErrorI18n(c, http.StatusBadRequest, "validation.invalidBody")
			return
		}
	}

	platform := service.Platform(strings.ToLower(strings.TrimSpace(c.PostForm("platform"))))
	switch platform {
	case service.PlatformIOS, service.PlatformAndroid, service.PlatformWeb:
	default:
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.invalidPlatform")
		return
	}

	version := strings.TrimSpace(c.PostForm("versionNumber"))
	if version == "" {
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.missingParams")
		return
	}

	code, err := strconv.Atoi(strings.TrimSpace(c.PostForm("versionCode")))
	if err != nil || code <= 0 {
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.invalidVersionCode")
		return
	}

	// Parsed strictly: anything that is not the literal "true"/"false" is refused
	// rather than coerced — Go's ParseBool reads "1"/"T" as true, and a CI script
	// sending "yes" would silently publish a non-mandatory release.
	var mandatory bool
	switch strings.TrimSpace(c.PostForm("isMandatoryUpdate")) {
	case "true":
		mandatory = true
	case "false":
		mandatory = false
	default:
		response.ErrorI18n(c, http.StatusBadRequest, "appVersion.invalidMandatoryFlag")
		return
	}

	in := service.ReleaseInput{
		Platform:     platform,
		Version:      version,
		VersionCode:  code,
		Mandatory:    mandatory,
		ReleaseNotes: strings.TrimSpace(c.PostForm("releaseNotes")),
	}

	// The artefact is OPTIONAL: ios/web have no APK here, and a policy-only
	// publish (raising the floor on an already-uploaded build) is legitimate.
	if fh, ferr := c.FormFile("file"); ferr == nil && fh != nil {
		f, openErr := fh.Open()
		if openErr != nil {
			response.ErrorI18n(c, http.StatusBadRequest, "appVersion.uploadFailed")
			return
		}
		defer f.Close()
		in.Artifact = f
		in.Filename = fh.Filename
		in.ContentType = fh.Header.Get("Content-Type")
	}

	av, err := h.svc.Release(c.Request.Context(), in)
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "appVersion.releaseSuccess", av)
}
