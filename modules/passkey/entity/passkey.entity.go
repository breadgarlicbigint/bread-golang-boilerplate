package entity

import (
	"time"

	"github.com/google/uuid"
)

// DeviceType identifies whether the credential authenticator is platform (biometric)
// or cross-platform (hardware key like YubiKey).
type AuthenticatorAttachment string

const (
	AttachmentPlatform      AuthenticatorAttachment = "platform"       // TouchID, FaceID, Windows Hello
	AttachmentCrossPlatform AuthenticatorAttachment = "cross-platform" // YubiKey, passkey manager
)

// Passkey stores a FIDO2/WebAuthn credential linked to a user.
// One user can have multiple passkeys (multiple devices).
type Passkey struct {
	ID                  uuid.UUID      `bson:"_id,omitempty"         json:"id"`
	UserID              uuid.UUID      `bson:"userId"                json:"userId"`
	TenantID            string                  `bson:"tenantId"              json:"tenantId,omitempty"`
	// WebAuthn fields
	CredentialID        []byte                  `bson:"credentialId"          json:"-"`
	CredentialIDBase64  string                  `bson:"credentialIdBase64"    json:"credentialId"` // URL-safe base64 for display
	PublicKey           []byte                  `bson:"publicKey"             json:"-"`
	AttestationType     string                  `bson:"attestationType"       json:"attestationType"`
	Transport           []string                `bson:"transport"             json:"transport"`
	Attachment          AuthenticatorAttachment `bson:"attachment"            json:"attachment"`
	SignCount           uint32                  `bson:"signCount"             json:"signCount"`
	// Metadata
	FriendlyName        string                  `bson:"friendlyName"          json:"friendlyName"` // user-given name e.g. "MacBook Touch ID"
	AAGUID              string                  `bson:"aaguid"                json:"aaguid"`       // authenticator model identifier
	BackedUp            bool                    `bson:"backedUp"              json:"backedUp"`     // passkey synced to cloud
	LastUsedAt          *time.Time              `bson:"lastUsedAt"            json:"lastUsedAt,omitempty"`
	CreatedAt           time.Time               `bson:"createdAt"             json:"createdAt"`
	UpdatedAt           time.Time               `bson:"updatedAt"             json:"updatedAt"`
}

func (p *Passkey) IsBiometric() bool {
	return p.Attachment == AttachmentPlatform
}
