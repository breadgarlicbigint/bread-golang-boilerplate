package entity

import (
	"time"

	"github.com/google/uuid"
)

// ── Notification types ────────────────────────────────────────────────────────

type NotifType string
type NotifChannel string
type NotifStatus string

const (
	// Types match the NestJS boilerplate's notification type enum
	TypeSystem       NotifType = "system"
	TypeAuth         NotifType = "auth"
	TypeUser         NotifType = "user"
	TypePromotion    NotifType = "promotion"
	TypeAlert        NotifType = "alert"
	TypeInfo         NotifType = "info"

	// Channels
	ChannelEmail     NotifChannel = "email"
	ChannelPush      NotifChannel = "push"
	ChannelInApp     NotifChannel = "in_app"
	ChannelSilent    NotifChannel = "silent"
	ChannelWhatsApp  NotifChannel = "whatsapp"
	ChannelSMS       NotifChannel = "sms"

	// Status
	StatusPending    NotifStatus = "pending"
	StatusSent       NotifStatus = "sent"
	StatusDelivered  NotifStatus = "delivered"
	StatusFailed     NotifStatus = "failed"
	StatusRead       NotifStatus = "read"
)

// Notification is a single in-app notification stored in MongoDB.
type Notification struct {
	ID          uuid.UUID     `bson:"_id,omitempty"    json:"id"`
	UserID      uuid.UUID     `bson:"userId"           json:"userId"`
	TenantID    string                 `bson:"tenantId"         json:"tenantId,omitempty"`
	Type        NotifType              `bson:"type"             json:"type"`
	Channel     NotifChannel           `bson:"channel"          json:"channel"`
	Status      NotifStatus            `bson:"status"           json:"status"`
	Title       string                 `bson:"title"            json:"title"`
	Body        string                 `bson:"body"             json:"body"`
	ImageURL    string                 `bson:"imageUrl"         json:"imageUrl,omitempty"`
	Data        map[string]interface{} `bson:"data"             json:"data,omitempty"`
	ActionURL   string                 `bson:"actionUrl"        json:"actionUrl,omitempty"`
	IsRead      bool                   `bson:"isRead"           json:"isRead"`
	ReadAt      *time.Time             `bson:"readAt"           json:"readAt,omitempty"`
	SentAt      *time.Time             `bson:"sentAt"           json:"sentAt,omitempty"`
	FailReason  string                 `bson:"failReason"       json:"failReason,omitempty"`
	CreatedAt   time.Time              `bson:"createdAt"        json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt"        json:"updatedAt"`
}

// ── Device token ──────────────────────────────────────────────────────────────

type DevicePlatform string

const (
	DevicePlatformIOS     DevicePlatform = "ios"
	DevicePlatformAndroid DevicePlatform = "android"
	DevicePlatformWeb     DevicePlatform = "web"
)

// DeviceToken stores an FCM push token linked to a user + device.
type DeviceToken struct {
	ID           uuid.UUID `bson:"_id,omitempty"   json:"id"`
	UserID       uuid.UUID `bson:"userId"          json:"userId"`
	TenantID     string             `bson:"tenantId"        json:"tenantId,omitempty"`
	Token        string             `bson:"token"           json:"token"`
	Platform     DevicePlatform     `bson:"platform"        json:"platform"`
	DeviceModel  string             `bson:"deviceModel"     json:"deviceModel,omitempty"`
	AppVersion   string             `bson:"appVersion"      json:"appVersion,omitempty"`
	IsActive     bool               `bson:"isActive"        json:"isActive"`
	LastSeenAt   time.Time          `bson:"lastSeenAt"      json:"lastSeenAt"`
	CreatedAt    time.Time          `bson:"createdAt"       json:"createdAt"`
}

// ── User notification preferences ────────────────────────────────────────────

// NotificationPreferences stores per-user channel + type opt-in/out settings.
// Mirrors the NestJS notification_preferences collection.
type NotificationPreferences struct {
	ID       uuid.UUID       `bson:"_id,omitempty"  json:"id"`
	UserID   uuid.UUID       `bson:"userId"         json:"userId"`
	TenantID string                   `bson:"tenantId"       json:"tenantId,omitempty"`
	// Per-channel enabled flags
	Channels map[string]bool          `bson:"channels"       json:"channels"` // key = NotifChannel
	// Per-type enabled flags per channel: types[typeKey][channelKey] = enabled
	Types    map[string]map[string]bool `bson:"types"         json:"types"`
	UpdatedAt time.Time                `bson:"updatedAt"      json:"updatedAt"`
}

func DefaultPreferences(userID uuid.UUID) NotificationPreferences {
	return NotificationPreferences{
		ID:     uuid.New(),
		UserID: userID,
		Channels: map[string]bool{
			string(ChannelEmail):    true,
			string(ChannelPush):     true,
			string(ChannelInApp):    true,
			string(ChannelSilent):   true,
			string(ChannelSMS):      false,
			string(ChannelWhatsApp): false,
		},
		Types: map[string]map[string]bool{
			string(TypeSystem):    {string(ChannelEmail): true, string(ChannelPush): true, string(ChannelInApp): true},
			string(TypeAuth):      {string(ChannelEmail): true, string(ChannelPush): true, string(ChannelInApp): true},
			string(TypeUser):      {string(ChannelEmail): true, string(ChannelPush): true, string(ChannelInApp): true},
			string(TypePromotion): {string(ChannelEmail): true, string(ChannelPush): false, string(ChannelInApp): true},
			string(TypeAlert):     {string(ChannelEmail): true, string(ChannelPush): true, string(ChannelInApp): true},
			string(TypeInfo):      {string(ChannelEmail): false, string(ChannelPush): true, string(ChannelInApp): true},
		},
		UpdatedAt: time.Now(),
	}
}
