package entity

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusBlocked  UserStatus = "blocked"
	UserStatusInactive UserStatus = "inactive"
)

// NotifSettings holds per-channel notification preferences stored on the user doc.
type NotifSettings struct {
	Email    bool `bson:"email"    json:"email"`
	Push     bool `bson:"push"     json:"push"`
	InApp    bool `bson:"inApp"    json:"inApp"`
	SMS      bool `bson:"sms"      json:"sms"`
	WhatsApp bool `bson:"whatsapp" json:"whatsapp"`
}

func DefaultNotifSettings() NotifSettings {
	return NotifSettings{Email: true, Push: true, InApp: true}
}

// User is the MongoDB document for user accounts.
type User struct {
	ID               uuid.UUID      `bson:"_id"                  json:"id"`
	Email            string         `bson:"email"                json:"email"`
	Username         string         `bson:"username"             json:"username"`
	PasswordHash     string         `bson:"passwordHash"         json:"-"`
	FirstName        string         `bson:"firstName"            json:"firstName"`
	LastName         string         `bson:"lastName"             json:"lastName"`
	ProfilePicture   string         `bson:"profilePicture"       json:"profilePicture,omitempty"`
	PhoneNumber      string         `bson:"phoneNumber"          json:"phoneNumber,omitempty"`
	Gender           string         `bson:"gender"               json:"gender,omitempty"`
	LastLoginAt      *time.Time     `bson:"lastLoginAt"          json:"lastLoginAt,omitempty"`
	Status           UserStatus     `bson:"status"               json:"status"`
	RoleID           uuid.UUID      `bson:"roleId"               json:"roleId"`
	EmailVerified    bool           `bson:"emailVerified"        json:"emailVerified"`
	TwoFAEnabled     bool           `bson:"twoFAEnabled"         json:"twoFAEnabled"`
	TwoFASecret      string         `bson:"twoFASecret"          json:"-"`
	BackupCodes      []string       `bson:"backupCodes"          json:"-"`
	GoogleID         string         `bson:"googleId"             json:"googleId,omitempty"`
	AppleID          string         `bson:"appleId"              json:"appleId,omitempty"`
	PasswordAttempts int            `bson:"passwordAttempts"     json:"-"`
	LockedUntil      *time.Time     `bson:"lockedUntil"          json:"-"`
	NotifSettings    NotifSettings  `bson:"notifSettings"        json:"notifSettings"`
	DeletedAt        *time.Time     `bson:"deletedAt"            json:"deletedAt,omitempty"`
	CreatedAt        time.Time      `bson:"createdAt"            json:"createdAt"`
	UpdatedAt        time.Time      `bson:"updatedAt"            json:"updatedAt"`
}

func (u *User) FullName() string { return u.FirstName + " " + u.LastName }
func (u *User) IsActive() bool   { return u.Status == UserStatusActive && u.DeletedAt == nil }
func (u *User) IsLocked() bool   { return u.LockedUntil != nil && time.Now().Before(*u.LockedUntil) }
