package service

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMSender sends push notifications via Firebase Cloud Messaging.
type FCMSender struct {
	client *messaging.Client
}

// NewFCMSender initialises a Firebase app from a service-account JSON file path
// or from GOOGLE_APPLICATION_CREDENTIALS env var (preferred for production).
func NewFCMSender(credentialFile string) (*FCMSender, error) {
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
	return &FCMSender{client: client}, nil
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
func (s *FCMSender) Send(ctx context.Context, p PushPayload) error {
	msg := &messaging.Message{
		Token: p.Token,
		Data:  p.Data,
		Notification: &messaging.Notification{
			Title:    p.Title,
			Body:     p.Body,
			ImageURL: p.ImageURL,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				Title:    p.Title,
				Body:     p.Body,
				ImageURL: p.ImageURL,
				Sound:    "default",
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: p.Title,
						Body:  p.Body,
					},
					Badge: &p.Badge,
					Sound: "default",
				},
			},
		},
	}

	if p.Silent {
		// Silent / data-only notification — no visible alert
		msg.Notification = nil
		msg.Android.Notification = nil
		msg.APNS.Payload.Aps.Alert = nil
		msg.APNS.Headers["apns-priority"] = "5"
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
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title:    p.Title,
			Body:     p.Body,
			ImageURL: p.ImageURL,
		},
		Data: p.Data,
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
