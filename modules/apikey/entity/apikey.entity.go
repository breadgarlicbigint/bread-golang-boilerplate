package entity

import (
	"time"

	"github.com/google/uuid"
)

type APIKeyType string

const (
	APIKeyTypePublic  APIKeyType = "public"
	APIKeyTypePrivate APIKeyType = "private"
)

// APIKey is stored in the `api_keys` collection.
// The raw key value is shown ONCE at creation; only its hash is persisted.
type APIKey struct {
	ID          uuid.UUID `bson:"_id,omitempty"   json:"id"`
	Name        string             `bson:"name"            json:"name"`
	Description string             `bson:"description"     json:"description,omitempty"`
	KeyHash     string             `bson:"keyHash"         json:"-"`     // bcrypt hash of raw key
	KeyPrefix   string             `bson:"keyPrefix"       json:"prefix"` // first 8 chars for lookup
	Type        APIKeyType         `bson:"type"            json:"type"`
	IsActive    bool               `bson:"isActive"        json:"isActive"`
	Permissions []string           `bson:"permissions"     json:"permissions"`
	ExpiresAt   *time.Time         `bson:"expiresAt"       json:"expiresAt,omitempty"`
	LastUsedAt  *time.Time         `bson:"lastUsedAt"      json:"lastUsedAt,omitempty"`
	CreatedBy   uuid.UUID `bson:"createdBy"       json:"createdBy"`
	CreatedAt   time.Time          `bson:"createdAt"       json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"       json:"updatedAt"`
}

func (k *APIKey) IsExpired() bool {
	return k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt)
}
