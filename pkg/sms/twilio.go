package sms

import (
	"context"
	"fmt"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"go.uber.org/zap"
)

// Channel determines which Twilio service to use.
type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

// Sender sends SMS and WhatsApp messages via Twilio.
type Sender struct {
	client          *twilio.RestClient
	accountSID      string
	fromSMS         string // e.g. "+15551234567"
	fromWhatsApp    string // e.g. "whatsapp:+14155238886"
	smsEnabled      bool
	whatsAppEnabled bool
	log             *zap.Logger
}

type Config struct {
	AccountSID   string
	AuthToken    string
	FromSMS      string
	FromWhatsApp string
	// SMSEnabled / WhatsAppEnabled turn each channel LIVE. A channel that is
	// disabled — or enabled but missing credentials — is simulated: the message
	// is logged and nothing leaves the process, so dev and tests run offline.
	SMSEnabled      bool
	WhatsAppEnabled bool
	Logger          *zap.Logger
}

func New(cfg Config) *Sender {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})
	log := cfg.Logger
	if log == nil {
		log = zap.NewNop()
	}
	return &Sender{
		client:          client,
		accountSID:      cfg.AccountSID,
		fromSMS:         cfg.FromSMS,
		fromWhatsApp:    cfg.FromWhatsApp,
		smsEnabled:      cfg.SMSEnabled,
		whatsAppEnabled: cfg.WhatsAppEnabled,
		log:             log,
	}
}

// live reports whether a channel should actually dispatch: its flag is set and
// the credentials it needs are present. Otherwise Send simulates.
func (s *Sender) live(channel Channel) bool {
	if s.accountSID == "" {
		return false
	}
	if channel == ChannelWhatsApp {
		return s.whatsAppEnabled && s.fromWhatsApp != ""
	}
	return s.smsEnabled && s.fromSMS != ""
}

// SendOTP sends a one-time password via SMS or WhatsApp.
func (s *Sender) SendOTP(ctx context.Context, to, code string, channel Channel) error {
	body := fmt.Sprintf("Your verification code is: %s\n\nThis code expires in 10 minutes. Do not share it with anyone.", code)
	return s.Send(ctx, to, body, channel)
}

// Send dispatches a raw message. When the channel is not live (disabled or
// unconfigured) it logs a "(sim)" line and returns nil, so callers see success
// and nothing leaves the process.
func (s *Sender) Send(_ context.Context, to, body string, channel Channel) error {
	if !s.live(channel) {
		s.log.Info("sms: (sim) message not sent — channel disabled or unconfigured",
			zap.String("channel", string(channel)), zap.String("to", to))
		return nil
	}

	from := s.fromSMS
	if channel == ChannelWhatsApp {
		from = s.fromWhatsApp
		to = "whatsapp:" + to
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(to)
	params.SetFrom(from)
	params.SetBody(body)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio %s: send to %s: %w", channel, to, err)
	}
	return nil
}
