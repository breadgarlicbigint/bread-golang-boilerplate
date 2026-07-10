package sms

import (
	"context"
	"fmt"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// Channel determines which Twilio service to use.
type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

// Sender sends SMS and WhatsApp messages via Twilio.
type Sender struct {
	client        *twilio.RestClient
	fromSMS       string // e.g. "+15551234567"
	fromWhatsApp  string // e.g. "whatsapp:+14155238886"
}

type Config struct {
	AccountSID   string
	AuthToken    string
	FromSMS      string
	FromWhatsApp string
}

func New(cfg Config) *Sender {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: cfg.AccountSID,
		Password: cfg.AuthToken,
	})
	return &Sender{
		client:       client,
		fromSMS:      cfg.FromSMS,
		fromWhatsApp: cfg.FromWhatsApp,
	}
}

// SendOTP sends a one-time password via SMS or WhatsApp.
func (s *Sender) SendOTP(ctx context.Context, to, code string, channel Channel) error {
	body := fmt.Sprintf("Your verification code is: %s\n\nThis code expires in 10 minutes. Do not share it with anyone.", code)
	return s.Send(ctx, to, body, channel)
}

// Send dispatches a raw message.
func (s *Sender) Send(_ context.Context, to, body string, channel Channel) error {
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
