package email

import (
	"context"
	"fmt"
)

// Message is a pre-built outgoing email (HTML + plain-text).
type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string
}

// Sender delivers a pre-built Message over some transport (SES, SMTP, ...).
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Mailer sends transactional email through whichever Sender it's built with —
// AWS SES or SMTP, selected at startup via MAIL_DRIVER (see NewMailerFromConfig).
type Mailer struct {
	sender Sender
}

// NewMailer wraps a Sender. Use NewSESSender or NewSMTPSender to build one,
// or NewMailerFromConfig to pick automatically based on MAIL_DRIVER.
func NewMailer(sender Sender) *Mailer {
	return &Mailer{sender: sender}
}

// Send dispatches a Message through the underlying Sender.
func (m *Mailer) Send(ctx context.Context, msg Message) error {
	return m.sender.Send(ctx, msg)
}

// ── Convenience senders (use React Email templates) ───────────────────────────

// SendVerifyEmail renders the verify-email React Email template and sends it.
func (m *Mailer) SendVerifyEmail(ctx context.Context, to, name, verifyLink, expireHrs string) error {
	r, err := RenderVerifyEmail(VerifyEmailData{
		Name:       name,
		VerifyLink: verifyLink,
		ExpireHrs:  expireHrs,
	})
	if err != nil {
		return err
	}
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: "Verify your email address",
		HTML:    r.HTML,
		Text:    r.Text,
	})
}

// SendPasswordReset renders the reset-password template and sends it.
func (m *Mailer) SendPasswordReset(ctx context.Context, to, name, resetLink, expireMins, ip string) error {
	r, err := RenderResetPassword(ResetPasswordData{
		Name:       name,
		ResetLink:  resetLink,
		ExpireMins: expireMins,
		IPAddress:  ip,
	})
	if err != nil {
		return err
	}
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: "Reset your password",
		HTML:    r.HTML,
		Text:    r.Text,
	})
}

// SendWelcome renders the welcome template and sends it.
func (m *Mailer) SendWelcome(ctx context.Context, to, name, dashboardURL, docsURL string) error {
	r, err := RenderWelcome(WelcomeData{
		Name:         name,
		DashboardURL: dashboardURL,
		DocsURL:      docsURL,
	})
	if err != nil {
		return err
	}
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: "Welcome to Bread Boilerplate 🎉",
		HTML:    r.HTML,
		Text:    r.Text,
	})
}

// SendOTPCode renders the otp-code template and sends it.
func (m *Mailer) SendOTPCode(ctx context.Context, to, name, code, expireMins, purpose string) error {
	r, err := RenderOTPCode(OTPCodeData{
		Name:       name,
		Code:       code,
		ExpireMins: expireMins,
		Purpose:    purpose,
	})
	if err != nil {
		return err
	}
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: fmt.Sprintf("Your %s code: %s", purpose, code),
		HTML:    r.HTML,
		Text:    r.Text,
	})
}

// SendNotification renders the generic notification template and sends it.
func (m *Mailer) SendNotification(ctx context.Context, to, name, title, body, ctaLabel, ctaURL string) error {
	r, err := RenderNotification(NotificationData{
		Name:     name,
		Title:    title,
		Body:     body,
		CTALabel: ctaLabel,
		CTAURL:   ctaURL,
	})
	if err != nil {
		return err
	}
	return m.Send(ctx, Message{
		To:      []string{to},
		Subject: title,
		HTML:    r.HTML,
		Text:    r.Text,
	})
}
