package email

import "embed"

// dist/ is populated by `make build-emails` (npm run build in email-templates/).
// The compiled HTML/text files are embedded at compile time so no Node.js is
// needed in production.

//go:embed dist/*.html dist/*.txt
var templateFS embed.FS

// ── Template names ────────────────────────────────────────────────────────────

type TemplateName string

const (
	TplVerifyEmail   TemplateName = "verify-email"
	TplResetPassword TemplateName = "reset-password"
	TplWelcome       TemplateName = "welcome"
	TplOTPCode       TemplateName = "otp-code"
	TplNotification  TemplateName = "notification"
)

// ── Renderer ──────────────────────────────────────────────────────────────────

// Rendered holds both the HTML and plain-text versions of an email.
type Rendered struct {
	HTML string
	Text string
}
