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
	TplVerifyEmail    TemplateName = "verify-email"
	TplResetPassword  TemplateName = "reset-password"
	TplWelcome        TemplateName = "welcome"
	TplOTPCode        TemplateName = "otp-code"
	TplNotification   TemplateName = "notification"
)

// ── Token maps ────────────────────────────────────────────────────────────────
// Each template has a fixed set of placeholder tokens that must be supplied.
// Tokens must match the constants defined in the .tsx files (P.xxx = "__XXX__").

// VerifyEmailData holds the tokens for verify-email template.
type VerifyEmailData struct {
	Name       string // __NAME__
	VerifyLink string // __VERIFY_LINK__
	ExpireHrs  string // __EXPIRE_HRS__  e.g. "24"
}

// ResetPasswordData holds the tokens for reset-password template.
type ResetPasswordData struct {
	Name        string // __NAME__
	ResetLink   string // __RESET_LINK__
	ExpireMins  string // __EXPIRE_MINS__  e.g. "60"
	IPAddress   string // __IP_ADDRESS__
}

// WelcomeData holds the tokens for welcome template.
type WelcomeData struct {
	Name         string // __NAME__
	DashboardURL string // __DASHBOARD_URL__
	DocsURL      string // __DOCS_URL__
}

// OTPCodeData holds the tokens for otp-code template.
type OTPCodeData struct {
	Name       string // __NAME__
	Code       string // __CODE__
	ExpireMins string // __EXPIRE_MINS__  e.g. "10"
	Purpose    string // __PURPOSE__  e.g. "Two-factor authentication"
}

// NotificationData holds the tokens for the generic notification template.
type NotificationData struct {
	Name     string // __NAME__
	Title    string // __TITLE__
	Body     string // __BODY__
	CTALabel string // __CTA_LABEL__
	CTAURL   string // __CTA_URL__
}

// ── Renderer ──────────────────────────────────────────────────────────────────

// Rendered holds both the HTML and plain-text versions of an email.
type Rendered struct {
	HTML string
	Text string
}

// ── Public render functions (use LocalizedMailer for full i18n support) ────────
// These functions use fixed __TOKEN__ placeholders only — no text translation.
// For fully localised emails, use LocalizedMailer.Render*/Send* instead.

func RenderVerifyEmail(d VerifyEmailData) (*Rendered, error) {
	return renderWithTokens(TplVerifyEmail, map[string]string{
		"__NAME__":        d.Name,
		"__VERIFY_LINK__": d.VerifyLink,
		"__EXPIRE_HRS__":  d.ExpireHrs,
	})
}

func RenderResetPassword(d ResetPasswordData) (*Rendered, error) {
	return renderWithTokens(TplResetPassword, map[string]string{
		"__NAME__":        d.Name,
		"__RESET_LINK__":  d.ResetLink,
		"__EXPIRE_MINS__": d.ExpireMins,
		"__IP_ADDRESS__":  d.IPAddress,
	})
}

func RenderWelcome(d WelcomeData) (*Rendered, error) {
	return renderWithTokens(TplWelcome, map[string]string{
		"__NAME__":          d.Name,
		"__DASHBOARD_URL__": d.DashboardURL,
		"__DOCS_URL__":      d.DocsURL,
	})
}

func RenderOTPCode(d OTPCodeData) (*Rendered, error) {
	return renderWithTokens(TplOTPCode, map[string]string{
		"__NAME__":        d.Name,
		"__CODE__":        d.Code,
		"__EXPIRE_MINS__": d.ExpireMins,
		"__PURPOSE__":     d.Purpose,
	})
}

func RenderNotification(d NotificationData) (*Rendered, error) {
	return renderWithTokens(TplNotification, map[string]string{
		"__NAME__":      d.Name,
		"__TITLE__":     d.Title,
		"__BODY__":      d.Body,
		"__CTA_LABEL__": d.CTALabel,
		"__CTA_URL__":   d.CTAURL,
	})
}
