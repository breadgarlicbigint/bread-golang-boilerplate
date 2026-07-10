package entity

import (
	"time"

	"github.com/google/uuid"
)

type Permission string
type RoleSlug string

const (
	RoleAdmin  RoleSlug = "admin"
	RoleUser   RoleSlug = "user"
	RoleMember RoleSlug = "member"

	PermAll      Permission = "*"
	PermUserRead Permission = "user:read"
)

type Role struct {
	ID          uuid.UUID    `bson:"_id"         json:"id"`
	Name        string       `bson:"name"        json:"name"`
	Slug        RoleSlug     `bson:"slug"        json:"slug"`
	Description string       `bson:"description" json:"description"`
	Permissions []Permission `bson:"permissions" json:"permissions"`
	IsSystem    bool         `bson:"isSystem"    json:"isSystem"`
	CreatedAt   time.Time    `bson:"createdAt"   json:"createdAt"`
	UpdatedAt   time.Time    `bson:"updatedAt"   json:"updatedAt"`
}
