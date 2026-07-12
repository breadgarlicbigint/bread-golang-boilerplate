package email

import (
	"context"
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
