package entity

import (
	"time"

	"github.com/google/uuid"
)

// UserMobile stores a verified phone number linked to a user.
type UserMobile struct {
	ID          uuid.UUID `bson:"_id,omitempty"   json:"id"`
	UserID      uuid.UUID `bson:"userId"          json:"userId"`
	TenantID    string             `bson:"tenantId"        json:"tenantId,omitempty"`
	CountryCode string             `bson:"countryCode"     json:"countryCode"` // e.g. "+1"
	Number      string             `bson:"number"          json:"number"`      // local number without country code
	E164        string             `bson:"e164"            json:"e164"`        // canonical form e.g. "+15551234567"
	IsVerified  bool               `bson:"isVerified"      json:"isVerified"`
	IsPrimary   bool               `bson:"isPrimary"       json:"isPrimary"`
	VerifiedAt  *time.Time         `bson:"verifiedAt"      json:"verifiedAt,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt"       json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"       json:"updatedAt"`
}

// MobileOTP is a short-lived verification code stored in Redis (not MongoDB).
// Key: otp:<e164>  Value: hashed OTP code  TTL: 10 minutes
type MobileOTP struct {
	E164      string    `json:"e164"`
	CodeHash  string    `json:"codeHash"`
	Attempts  int       `json:"attempts"`
	ExpiresAt time.Time `json:"expiresAt"`
}
