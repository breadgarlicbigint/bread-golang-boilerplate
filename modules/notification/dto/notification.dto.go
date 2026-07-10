package dto

import "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/entity"

// ── Send ──────────────────────────────────────────────────────────────────────

// SendRequest is used by internal services to dispatch a notification.
type SendRequest struct {
	UserID    string                 `json:"userId"    validate:"required"`
	Type      entity.NotifType       `json:"type"      validate:"required"`
	Channel   entity.NotifChannel    `json:"channel"   validate:"required"`
	Title     string                 `json:"title"     validate:"required"`
	Body      string                 `json:"body"      validate:"required"`
	ImageURL  string                 `json:"imageUrl"`
	Data      map[string]interface{} `json:"data"`
	ActionURL string                 `json:"actionUrl"`
}

// BroadcastRequest sends to multiple users (admin use-case).
type BroadcastRequest struct {
	UserIDs  []string               `json:"userIds"  validate:"required,min=1"`
	Type     entity.NotifType       `json:"type"     validate:"required"`
	Channel  entity.NotifChannel    `json:"channel"  validate:"required"`
	Title    string                 `json:"title"    validate:"required"`
	Body     string                 `json:"body"     validate:"required"`
	Data     map[string]interface{} `json:"data"`
}

// ── Preferences ───────────────────────────────────────────────────────────────

type UpdatePreferencesRequest struct {
	Channels map[string]bool            `json:"channels"`
	Types    map[string]map[string]bool `json:"types"`
}

// ── Device Token ──────────────────────────────────────────────────────────────

type RegisterDeviceRequest struct {
	Token        string                 `json:"token"        validate:"required"`
	Platform     entity.DevicePlatform  `json:"platform"     validate:"required,oneof=ios android web"`
	DeviceModel  string                 `json:"deviceModel"`
	AppVersion   string                 `json:"appVersion"`
}

// ── Response ──────────────────────────────────────────────────────────────────

type NotificationResponse struct {
	ID        string                 `json:"id"`
	Type      entity.NotifType       `json:"type"`
	Channel   entity.NotifChannel    `json:"channel"`
	Status    entity.NotifStatus     `json:"status"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	ImageURL  string                 `json:"imageUrl,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	ActionURL string                 `json:"actionUrl,omitempty"`
	IsRead    bool                   `json:"isRead"`
	CreatedAt string                 `json:"createdAt"`
}

type UnreadCountResponse struct {
	Unread int64 `json:"unread"`
}
