package entity

import (
	"time"

	"github.com/google/uuid"
)

type TenantStatus string
type TenantPlan string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusTrial     TenantStatus = "trial"

	TenantPlanFree       TenantPlan = "free"
	TenantPlanStarter    TenantPlan = "starter"
	TenantPlanPro        TenantPlan = "pro"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

// Tenant represents an organisation/workspace in a multi-tenant deployment.
type Tenant struct {
	ID           uuid.UUID `bson:"_id,omitempty"   json:"id"`
	Slug         string             `bson:"slug"            json:"slug"`   // URL-safe, unique, e.g. "acme-corp"
	Name         string             `bson:"name"            json:"name"`
	Domain       string             `bson:"domain"          json:"domain,omitempty"` // custom domain
	Status       TenantStatus       `bson:"status"          json:"status"`
	Plan         TenantPlan         `bson:"plan"            json:"plan"`
	OwnerID      uuid.UUID `bson:"ownerId"         json:"ownerId"`
	MaxUsers     int                `bson:"maxUsers"        json:"maxUsers"`
	Settings     TenantSettings     `bson:"settings"        json:"settings"`
	TrialEndsAt  *time.Time         `bson:"trialEndsAt"     json:"trialEndsAt,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt"       json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"       json:"updatedAt"`
	DeletedAt    *time.Time         `bson:"deletedAt"       json:"deletedAt,omitempty"`
}

// TenantSettings stores per-tenant feature toggles and customisation.
type TenantSettings struct {
	AllowPublicSignup  bool   `bson:"allowPublicSignup"  json:"allowPublicSignup"`
	RequireEmailVerify bool   `bson:"requireEmailVerify" json:"requireEmailVerify"`
	Require2FA         bool   `bson:"require2FA"         json:"require2FA"`
	MaxSessionsPerUser int    `bson:"maxSessionsPerUser" json:"maxSessionsPerUser"`
	LogoURL            string `bson:"logoUrl"            json:"logoUrl,omitempty"`
	PrimaryColor       string `bson:"primaryColor"       json:"primaryColor,omitempty"`
	SupportEmail       string `bson:"supportEmail"       json:"supportEmail,omitempty"`
}

// TenantMember links a user to a tenant with a specific role override.
type TenantMember struct {
	ID        uuid.UUID `bson:"_id,omitempty"  json:"id"`
	TenantID  uuid.UUID `bson:"tenantId"       json:"tenantId"`
	UserID    uuid.UUID `bson:"userId"         json:"userId"`
	Role      string             `bson:"role"           json:"role"` // tenant-scoped role
	JoinedAt  time.Time          `bson:"joinedAt"       json:"joinedAt"`
}
