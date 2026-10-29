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

	// Generic per-action permissions an admin-configurable role (or a per-user
	// grant) can draw from. The admin role itself holds PermAll ("*"). These are
	// deliberately generic examples — a real project renames/extends them.
	PermUsersRead      Permission = "users:read"
	PermUsersWrite     Permission = "users:write"
	PermAuditRead      Permission = "audit:read"
	PermSettingsManage Permission = "settings:manage"
)

// AllAssignablePermissions is the full per-action set a configurable role or a
// per-user grant may reference. The wildcard "*" is NOT in it: it is a claim
// that every gate passes, not an enumerable permission, so it is never grantable
// per user. shared perm_labels_test asserts every slug here has a label.
func AllAssignablePermissions() []Permission {
	return []Permission{
		PermUsersRead, PermUsersWrite, PermAuditRead, PermSettingsManage,
	}
}

// IsGrantablePermission reports whether slug may be granted/revoked per user.
// Unknown slugs and the wildcard are refused.
func IsGrantablePermission(slug string) bool {
	for _, p := range AllAssignablePermissions() {
		if string(p) == slug {
			return true
		}
	}
	return false
}

// AssignableTenantRoles is the set of roles a non-wildcard admin may assign when
// creating a user. It excludes `admin` — a tenant admin must not be able to mint
// a wildcard user. A wildcard holder (platform/super admin) bypasses this.
func AssignableTenantRoles() []RoleSlug {
	return []RoleSlug{RoleUser, RoleMember}
}

// IsAssignableTenantRole reports whether a non-wildcard admin may assign slug.
func IsAssignableTenantRole(slug RoleSlug) bool {
	for _, s := range AssignableTenantRoles() {
		if s == slug {
			return true
		}
	}
	return false
}

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
