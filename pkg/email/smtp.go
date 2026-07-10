package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// SMTPSender delivers Messages via a generic SMTP server — a self-hosted
// relay (Postfix), a local dev catcher (MailHog/Mailpit), or a provider's
// SMTP endpoint (SendGrid, Mailgun, Amazon SES SMTP interface, etc.).
//
// Port 465 is treated as implicit TLS; every other port goes through
// net/smtp.SendMail, which negotiates STARTTLS automatically when the
// server advertises it (587 and most others) and falls back to plaintext
// for unauthenticated local dev servers.
type SMTPSender struct {
	host      string
	port      string
	username  string
	password  string
	fromEmail string
	fromName  string
}

// NewSMTPSender builds a Sender backed by SMTP.
func NewSMTPSender(cfg config.SMTPConfig) *SMTPSender {
	return &SMTPSender{
		host:      cfg.Host,
		port:      cfg.Port,
		username:  cfg.Username,
		password:  cfg.Password,
		fromEmail: cfg.FromEmail,
		fromName:  cfg.FromName,
	}
}

// Send dispatches a Message via SMTP.
func (s *SMTPSender) Send(_ context.Context, msg Message) error {
	raw, err := buildMIMEMessage(s.fromName, s.fromEmail, msg)
	if err != nil {
		return fmt.Errorf("smtp: build message: %w", err)
	}

	addr := net.JoinHostPort(s.host, s.port)
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	if s.port == "465" {
		return s.sendImplicitTLS(addr, auth, msg.To, raw)
	}
	return smtp.SendMail(addr, auth, s.fromEmail, msg.To, raw)
}

// sendImplicitTLS handles port 465, which expects TLS from the first byte
// (net/smtp.SendMail only knows how to upgrade via STARTTLS, not this).
func (s *SMTPSender) sendImplicitTLS(addr string, auth smtp.Auth, to []string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return fmt.Errorf("smtp: tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer c.Close()

	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}
	if err := c.Mail(s.fromEmail); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

// buildMIMEMessage builds an RFC 822 multipart/alternative (text + HTML) message.
func buildMIMEMessage(fromName, fromEmail string, msg Message) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	var out bytes.Buffer
	fmt.Fprintf(&out, "From: %s <%s>\r\n", fromName, fromEmail)
	fmt.Fprintf(&out, "To: %s\r\n", strings.Join(msg.To, ", "))
	fmt.Fprintf(&out, "Subject: %s\r\n", mime.QEncoding.Encode("UTF-8", msg.Subject))
	out.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&out, "Content-Type: multipart/alternative; boundary=%s\r\n", writer.Boundary())
	out.WriteString("\r\n")

	textPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/plain; charset=UTF-8"}})
	if err != nil {
		return nil, err
	}
	if _, err := textPart.Write([]byte(msg.Text)); err != nil {
		return nil, err
	}

	htmlPart, err := writer.CreatePart(textproto.MIMEHeader{"Content-Type": {"text/html; charset=UTF-8"}})
	if err != nil {
		return nil, err
	}
	if _, err := htmlPart.Write([]byte(msg.HTML)); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	out.Write(body.Bytes())
	return out.Bytes(), nil
}
