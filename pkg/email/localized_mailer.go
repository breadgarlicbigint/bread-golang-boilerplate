package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	pkgi18n "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
)

// LocalizedMailer wraps Mailer and resolves all email text from the i18n
// system before dispatching. Every visible string in every template is
// translated. Mailer may be nil — Render* methods work without it.
type LocalizedMailer struct {
	mailer *Mailer
	tr     *pkgi18n.Translator
}

// NewLocalizedMailer creates a mailer that sends translated emails.
// Pass nil for mailer to use only the Render* methods (useful in tests).
func NewLocalizedMailer(mailer *Mailer, tr *pkgi18n.Translator) *LocalizedMailer {
	return &LocalizedMailer{mailer: mailer, tr: tr}
}

// ── Verify Email ──────────────────────────────────────────────────────────────

// RenderVerifyEmail builds the translated HTML+text for the verify-email template.
func (m *LocalizedMailer) RenderVerifyEmail(lang, name, verifyLink, expireHrs string) (*Rendered, error) {
	return renderWithTokens(TplVerifyEmail, mergeTokens(
		m.layoutTokens(lang),
		map[string]string{
			"__NAME__":           name,
			"__VERIFY_LINK__":    verifyLink,
			"__EXPIRE_HRS__":     expireHrs,
			"__PREVIEW__":        m.interpolateKey(lang, "email.verifyEmail.preview", "name", name),
			"__HEADING__":        m.t(lang, "email.verifyEmail.heading"),
			"__GREETING__":       m.interpolateKey(lang, "email.verifyEmail.greeting", "name", name),
			"__BODY__":           m.t(lang, "email.verifyEmail.body"),
			"__CTA_LABEL__":      m.t(lang, "email.verifyEmail.ctaLabel"),
			"__COPY_LINK_TEXT__": m.t(lang, "email.verifyEmail.copyLinkText"),
			"__EXPIRE_NOTE__":    m.interpolateKey(lang, "email.verifyEmail.expireNote", "expireHrs", expireHrs),
			"__IGNORE_NOTE__":    m.t(lang, "email.verifyEmail.ignoreNote"),
		},
	))
}

// SubjectVerifyEmail returns the translated email subject.
func (m *LocalizedMailer) SubjectVerifyEmail(lang, name string) string {
	return m.interpolateKey(lang, "email.verifyEmail.subject", "name", name)
}

// SendVerifyEmail renders and dispatches the verify-email template.
func (m *LocalizedMailer) SendVerifyEmail(ctx context.Context, lang, to, name, verifyLink, expireHrs string) error {
	r, err := m.RenderVerifyEmail(lang, name, verifyLink, expireHrs)
	if err != nil {
		return err
	}
	return m.send(ctx, to, m.SubjectVerifyEmail(lang, name), r)
}

// ── Reset Password ────────────────────────────────────────────────────────────

func (m *LocalizedMailer) RenderResetPassword(lang, name, resetLink, expireMins, ipAddress string) (*Rendered, error) {
	return renderWithTokens(TplResetPassword, mergeTokens(
		m.layoutTokens(lang),
		map[string]string{
			"__NAME__":           name,
			"__RESET_LINK__":     resetLink,
			"__EXPIRE_MINS__":    expireMins,
			"__IP_ADDRESS__":     ipAddress,
			"__PREVIEW__":        m.interpolateKey(lang, "email.resetPassword.preview", "name", name),
			"__HEADING__":        m.t(lang, "email.resetPassword.heading"),
			"__GREETING__":       m.interpolateKey(lang, "email.resetPassword.greeting", "name", name),
			"__BODY__":           m.t(lang, "email.resetPassword.body"),
			"__CTA_LABEL__":      m.t(lang, "email.resetPassword.ctaLabel"),
			"__COPY_LINK_TEXT__": m.t(lang, "email.resetPassword.copyLinkText"),
			"__SECURITY_TITLE__": m.t(lang, "email.resetPassword.securityTitle"),
			"__SECURITY_NOTE__": m.multi(lang, "email.resetPassword.securityNote",
				"expireMins", expireMins, "ipAddress", ipAddress),
			"__IGNORE_NOTE__": m.t(lang, "email.resetPassword.ignoreNote"),
		},
	))
}

func (m *LocalizedMailer) SubjectResetPassword(lang string) string {
	return m.t(lang, "email.resetPassword.subject")
}

func (m *LocalizedMailer) SendPasswordReset(ctx context.Context, lang, to, name, resetLink, expireMins, ipAddress string) error {
	r, err := m.RenderResetPassword(lang, name, resetLink, expireMins, ipAddress)
	if err != nil {
		return err
	}
	return m.send(ctx, to, m.SubjectResetPassword(lang), r)
}

// ── Welcome ───────────────────────────────────────────────────────────────────

func (m *LocalizedMailer) RenderWelcome(lang, name, dashboardURL, docsURL string) (*Rendered, error) {
	return renderWithTokens(TplWelcome, mergeTokens(
		m.layoutTokens(lang),
		map[string]string{
			"__NAME__":          name,
			"__DASHBOARD_URL__": dashboardURL,
			"__DOCS_URL__":      docsURL,
			"__PREVIEW__":       m.interpolateKey(lang, "email.welcome.preview", "name", name),
			"__HEADING__":       m.t(lang, "email.welcome.heading"),
			"__GREETING__":      m.interpolateKey(lang, "email.welcome.greeting", "name", name),
			"__SUBHEADING__":    m.t(lang, "email.welcome.subheading"),
			"__CTA_PRIMARY__":   m.t(lang, "email.welcome.ctaPrimary"),
			"__CTA_SECONDARY__": m.t(lang, "email.welcome.ctaSecondary"),
			"__HELP_NOTE__":     m.t(lang, "email.welcome.helpNote"),
			"__FEAT1_ICON__":    m.t(lang, "email.welcome.feat1Icon"),
			"__FEAT1_TITLE__":   m.t(lang, "email.welcome.feat1Title"),
			"__FEAT1_DESC__":    m.t(lang, "email.welcome.feat1Desc"),
			"__FEAT2_ICON__":    m.t(lang, "email.welcome.feat2Icon"),
			"__FEAT2_TITLE__":   m.t(lang, "email.welcome.feat2Title"),
			"__FEAT2_DESC__":    m.t(lang, "email.welcome.feat2Desc"),
			"__FEAT3_ICON__":    m.t(lang, "email.welcome.feat3Icon"),
			"__FEAT3_TITLE__":   m.t(lang, "email.welcome.feat3Title"),
			"__FEAT3_DESC__":    m.t(lang, "email.welcome.feat3Desc"),
		},
	))
}

func (m *LocalizedMailer) SubjectWelcome(lang, name string) string {
	return m.interpolateKey(lang, "email.welcome.subject", "name", name)
}

func (m *LocalizedMailer) SendWelcome(ctx context.Context, lang, to, name, dashboardURL, docsURL string) error {
	r, err := m.RenderWelcome(lang, name, dashboardURL, docsURL)
	if err != nil {
		return err
	}
	return m.send(ctx, to, m.SubjectWelcome(lang, name), r)
}

// ── OTP Code ──────────────────────────────────────────────────────────────────

func (m *LocalizedMailer) RenderOTPCode(lang, name, code, expireMins, purpose string) (*Rendered, error) {
	return renderWithTokens(TplOTPCode, mergeTokens(
		m.layoutTokens(lang),
		map[string]string{
			"__NAME__":          name,
			"__CODE__":          code,
			"__EXPIRE_MINS__":   expireMins,
			"__PURPOSE__":       purpose,
			"__PREVIEW__":       m.multi(lang, "email.otpCode.preview", "purpose", purpose, "code", code),
			"__HEADING__":       m.t(lang, "email.otpCode.heading"),
			"__GREETING__":      m.multi(lang, "email.otpCode.greeting", "name", name, "purpose", purpose),
			"__EXPIRE_NOTE__":   m.interpolateKey(lang, "email.otpCode.expireNote", "expireMins", expireMins),
			"__WARNING_TITLE__": m.t(lang, "email.otpCode.warningTitle"),
			"__WARNING_BODY__":  m.t(lang, "email.otpCode.warningBody"),
		},
	))
}

func (m *LocalizedMailer) SubjectOTPCode(lang, purpose, code string) string {
	return m.multi(lang, "email.otpCode.subject", "purpose", purpose, "code", code)
}

func (m *LocalizedMailer) SendOTPCode(ctx context.Context, lang, to, name, code, expireMins, purpose string) error {
	r, err := m.RenderOTPCode(lang, name, code, expireMins, purpose)
	if err != nil {
		return err
	}
	return m.send(ctx, to, m.SubjectOTPCode(lang, purpose, code), r)
}

// ── Notification ──────────────────────────────────────────────────────────────

func (m *LocalizedMailer) RenderNotification(lang, name, title, body, ctaLabel, ctaURL string) (*Rendered, error) {
	return renderWithTokens(TplNotification, mergeTokens(
		m.layoutTokens(lang),
		map[string]string{
			"__NAME__":      name,
			"__TITLE__":     title,
			"__BODY__":      body,
			"__CTA_LABEL__": ctaLabel,
			"__CTA_URL__":   ctaURL,
			"__PREVIEW__":   title,
			"__GREETING__":  m.interpolateKey(lang, "email.notification.greeting", "name", name),
		},
	))
}

func (m *LocalizedMailer) SendNotification(ctx context.Context, lang, to, name, title, body, ctaLabel, ctaURL string) error {
	r, err := m.RenderNotification(lang, name, title, body, ctaLabel, ctaURL)
	if err != nil {
		return err
	}
	return m.send(ctx, to, title, r)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// t translates a single key.
func (m *LocalizedMailer) t(lang, key string) string {
	return m.tr.T(lang, key)
}

// interpolateKey translates a key then substitutes one {placeholder}.
func (m *LocalizedMailer) interpolateKey(lang, key, varKey, varVal string) string {
	return strings.ReplaceAll(m.t(lang, key), "{"+varKey+"}", varVal)
}

// multi translates a key then substitutes multiple {placeholders} in order.
// vars must be "key1","val1","key2","val2",... pairs.
func (m *LocalizedMailer) multi(lang, key string, vars ...string) string {
	s := m.t(lang, key)
	for i := 0; i+1 < len(vars); i += 2 {
		s = strings.ReplaceAll(s, "{"+vars[i]+"}", vars[i+1])
	}
	return s
}

// layoutTokens returns shared layout token map with year filled in.
func (m *LocalizedMailer) layoutTokens(lang string) map[string]string {
	year := fmt.Sprintf("%d", time.Now().Year())
	return map[string]string{
		"__BRAND_NAME__":       m.t(lang, "email.layout.brandName"),
		"__FOOTER_COPYRIGHT__": strings.ReplaceAll(m.t(lang, "email.layout.footerCopyright"), "{year}", year),
		"__FOOTER_IGNORE__":    m.t(lang, "email.layout.footerIgnore"),
	}
}

// send dispatches a pre-rendered message via the underlying Mailer.
// Returns nil if mailer is nil (useful in tests / dry-run).
func (m *LocalizedMailer) send(ctx context.Context, to, subject string, r *Rendered) error {
	if m.mailer == nil {
		return nil
	}
	return m.mailer.Send(ctx, Message{To: []string{to}, Subject: subject, HTML: r.HTML, Text: r.Text})
}

// mergeTokens merges multiple token maps; later maps override earlier ones.
func mergeTokens(maps ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, mp := range maps {
		for k, v := range mp {
			out[k] = v
		}
	}
	return out
}

// renderWithTokens loads a template from the embedded FS and replaces all tokens.
func renderWithTokens(name TemplateName, tokens map[string]string) (*Rendered, error) {
	pairs := make([]string, 0, len(tokens)*2)
	for k, v := range tokens {
		pairs = append(pairs, k, v)
	}
	r := strings.NewReplacer(pairs...)

	htmlBytes, err := templateFS.ReadFile(fmt.Sprintf("dist/%s.html", name))
	if err != nil {
		return nil, fmt.Errorf("email: load %s.html: %w", name, err)
	}
	textBytes, err := templateFS.ReadFile(fmt.Sprintf("dist/%s.txt", name))
	if err != nil {
		return nil, fmt.Errorf("email: load %s.txt: %w", name, err)
	}

	return &Rendered{
		HTML: r.Replace(string(htmlBytes)),
		Text: r.Replace(string(textBytes)),
	}, nil
}
