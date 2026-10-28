package sms

import (
	"context"
	"testing"
)

// A channel that is disabled, or enabled but missing credentials, must be
// simulated: Send returns nil and makes no Twilio call, so callers succeed and
// nothing leaves the process offline.
func TestSend_SimulatesWhenNotLive(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"disabled with creds", Config{AccountSID: "AC1", FromSMS: "+15550000000", SMSEnabled: false}},
		{"enabled without creds", Config{SMSEnabled: true}},
		{"whatsapp disabled", Config{AccountSID: "AC1", FromWhatsApp: "whatsapp:+1", WhatsAppEnabled: false}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := New(c.cfg)
			ch := ChannelSMS
			if c.cfg.WhatsAppEnabled || c.cfg.FromWhatsApp != "" {
				ch = ChannelWhatsApp
			}
			if err := s.Send(context.Background(), "+15551234567", "hello", ch); err != nil {
				t.Errorf("simulated send should return nil, got %v", err)
			}
		})
	}
}

// live() requires both the flag and the credentials the channel needs.
func TestLive_RequiresFlagAndCredentials(t *testing.T) {
	if New(Config{AccountSID: "AC1", FromSMS: "+1", SMSEnabled: true}).live(ChannelSMS) != true {
		t.Error("SMS should be live with flag + creds")
	}
	if New(Config{AccountSID: "AC1", FromSMS: "+1", SMSEnabled: false}).live(ChannelSMS) {
		t.Error("SMS must not be live without the flag")
	}
	if New(Config{SMSEnabled: true, FromSMS: "+1"}).live(ChannelSMS) {
		t.Error("SMS must not be live without an account SID")
	}
}
