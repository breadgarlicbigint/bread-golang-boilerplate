// Package notification provides Multi-channel notification delivery (FCM, email, in-app, SMS).
//
// # Monolith usage
//
// Import the concrete service directly from modules/notification/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package notification

import "context"

// Service is the public contract other modules use.
type Service interface {
	Send(ctx context.Context, req SendRequest) error
}

// SendRequest is the minimal payload needed to dispatch a notification.
type SendRequest struct {
	UserID   string
	Type     string // "system" | "auth" | "user" | "promotion" | "alert" | "info"
	Channel  string // "email" | "push" | "in_app" | "silent" | "sms" | "whatsapp"
	Title    string
	Body     string
	Data     map[string]interface{}
}
