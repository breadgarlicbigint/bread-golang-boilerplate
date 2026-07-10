package email

import (
	"go.uber.org/zap"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// NewMailerFromConfig builds a Mailer using the transport selected by
// MAIL_DRIVER — "ses" (default) or "smtp". Returns nil if the selected
// driver isn't configured (no AWS credentials / no SMTP host) or fails to
// initialize; callers already treat a nil Mailer as "email sending disabled".
func NewMailerFromConfig(cfg *config.Config, log *zap.Logger) *Mailer {
	switch cfg.Mail.Driver {
	case "smtp":
		if cfg.Mail.SMTP.Host == "" {
			return nil
		}
		return NewMailer(NewSMTPSender(cfg.Mail.SMTP))
	default: // "ses"
		if cfg.AWS.AccessKeyID == "" {
			return nil
		}
		sender, err := NewSESSender(cfg.AWS)
		if err != nil {
			if log != nil {
				log.Warn("email: SES sender init failed", zap.Error(err))
			}
			return nil
		}
		return NewMailer(sender)
	}
}
