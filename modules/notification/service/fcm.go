package service

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// DefaultAndroidChannelID is used when no channel id is configured. The value
// MUST match `default_notification_channel_id` in the app's strings.xml and the
// channel the app creates at IMPORTANCE_HIGH — a payload naming a channel the
// device does not have is delivered on Firebase's own fallback channel at
// DEFAULT importance, i.e. silently into the tray with no heads-up alert.
// Nothing errors, so the symptom is "push doesn't work" from a path that is in
// fact delivering. Override per project with FCM_ANDROID_CHANNEL_ID.
const DefaultAndroidChannelID = "default"

// FCMSender sends push notifications via Firebase Cloud Messaging.
type FCMSender struct {
	client    *messaging.Client
	channelID string
}

// NewFCMSender initialises a Firebase app from a service-account JSON file path
// or from GOOGLE_APPLICATION_CREDENTIALS env var (preferred for production).
// channelID is the Android notification channel every push is delivered on; an
// empty value falls back to DefaultAndroidChannelID.
func NewFCMSender(credentialFile, channelID string) (*FCMSender, error) {
	var opts []option.ClientOption
	if credentialFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialFile))
	}

	app, err := firebase.NewApp(context.Background(), nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("fcm: init firebase app: %w", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("fcm: get messaging client: %w", err)
	}
	if channelID == "" {
		channelID = DefaultAndroidChannelID
	}
	return &FCMSender{client: client, channelID: channelID}, nil
}

// notification is the cross-platform alert block, absent for a silent message.
func notification(p PushPayload) *messaging.Notification {
	if p.Silent {
		return nil
	}
	return &messaging.Notification{Title: p.Title, Body: p.Body, ImageURL: p.ImageURL}
}

// androidConfig is the Android half of a message, shared by Send and
// SendMulticast rather than inlined in each: SendMulticast (the path production
// uses) carried no AndroidConfig at all until this existed, so high priority and
// the channel were configured only on the path nothing calls.
func androidConfig(p PushPayload, channelID string) *messaging.AndroidConfig {
	c := &messaging.AndroidConfig{Priority: "high"}
	if p.Silent {
		return c
	}
	c.Notification = &messaging.AndroidNotification{
		Title:     p.Title,
		Body:      p.Body,
		ImageURL:  p.ImageURL,
		Sound:     "default",
		ChannelID: channelID,
	}
	return c
}

// apnsConfig is the iOS half. Priority 5 for a silent message is APNs' own rule,
// not a preference: a background push sent at priority 10 is rejected.
func apnsConfig(p PushPayload) *messaging.APNSConfig {
	if p.Silent {
		return &messaging.APNSConfig{
			Headers: map[string]string{"apns-priority": "5"},
			Payload: &messaging.APNSPayload{Aps: &messaging.Aps{ContentAvailable: true}},
		}
	}
	badge := p.Badge
	return &messaging.APNSConfig{
		Headers: map[string]string{"apns-priority": "10"},
		Payload: &messaging.APNSPayload{
			Aps: &messaging.Aps{
				Alert: &messaging.ApsAlert{Title: p.Title, Body: p.Body},
				Badge: &badge,
				Sound: "default",
			},
		},
	}
}

// PushPayload contains all data needed for a single push message.
type PushPayload struct {
	Token    string                 // target FCM token
	Title    string
	Body     string
	ImageURL string
	Data     map[string]string      // custom key-value pairs
	// iOS-specific
	Badge    int
	Sound    string
	// Silent notification (no alert shown)
	Silent   bool
}

// Send dispatches a push notification to a single device token.
//
// Everything below the token is built by the same androidConfig/apnsConfig
// SendMulticast uses, so the two paths cannot drift on priority, channel or the
// silent-message rules — which they had, the multicast one carrying no platform
// config at all.
func (s *FCMSender) Send(ctx context.Context, p PushPayload) error {
	msg := &messaging.Message{
		Token:        p.Token,
		Data:         p.Data,
		Notification: notification(p),
		Android:      androidConfig(p, s.channelID),
		APNS:         apnsConfig(p),
	}

	if p.Silent {
		// A data-only message must carry no `notification` block at all — one
		// present is what makes the OS draw an alert, whatever the priority says.
		if msg.Data == nil {
			msg.Data = make(map[string]string)
		}
		msg.Data["silent"] = "true"
	}

	_, err := s.client.Send(ctx, msg)
	return err
}

// SendMulticast dispatches to up to 500 tokens at once.
// Returns the count of successful sends and any tokens that should be removed.
func (s *FCMSender) SendMulticast(ctx context.Context, tokens []string, p PushPayload) (int, []string, error) {
	if len(tokens) == 0 {
		return 0, nil, nil
	}

	msg := &messaging.MulticastMessage{
		Tokens:       tokens,
		Notification: notification(p),
		Data:         p.Data,
		Android:      androidConfig(p, s.channelID),
		APNS:         apnsConfig(p),
	}

	result, err := s.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		return 0, nil, fmt.Errorf("fcm: multicast: %w", err)
	}

	// Collect tokens that are no longer valid (unregistered or invalid)
	var staleTokens []string
	for i, r := range result.Responses {
		if !r.Success {
			code := r.Error.Error()
			if isStaleToken(code) {
				staleTokens = append(staleTokens, tokens[i])
			}
		}
	}

	return result.SuccessCount, staleTokens, nil
}

func isStaleToken(errMsg string) bool {
	staleErrors := []string{
		"registration-token-not-registered",
		"invalid-registration-token",
		"invalid-argument",
	}
	for _, e := range staleErrors {
		if contains(errMsg, e) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
