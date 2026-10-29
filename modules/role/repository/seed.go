package repository

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// RoleSeed defines a seeded, system role.
type RoleSeed struct {
	Slug        entity.RoleSlug
	Name        string
	Description string
	Permissions []entity.Permission
}

// SystemRoles are the roles seeded on boot. Admin holds the wildcard; the others
// start from a per-action set an admin can tailor per user via extra/revoked
// permissions. This is the single source of truth — both EnsureRoles (startup)
// and scripts/seed use it, so the two cannot drift.
func SystemRoles() []RoleSeed {
	return []RoleSeed{
		{entity.RoleAdmin, "Admin", "Full access", []entity.Permission{entity.PermAll}},
		{entity.RoleUser, "User", "Standard user", []entity.Permission{entity.PermUsersRead}},
		{entity.RoleMember, "Member", "Limited member", []entity.Permission{entity.PermUsersRead}},
	}
}

// EnsureRoles idempotently upserts the system roles and the unique slug index.
// Safe to run on every boot: permission sets are refreshed from SystemRoles
// (so a role gains a permission a new feature adds), while _id/createdAt are set
// only on insert so a role's identity and references survive.
func EnsureRoles(ctx context.Context) func(*RoleRepository) error {
	return func(r *RoleRepository) error { return r.EnsureRoles(ctx) }
}

// EnsureRoles idempotently upserts the system roles. See SystemRoles.
func (r *RoleRepository) EnsureRoles(ctx context.Context) error {
	if _, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "slug", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("idx_role_slug"),
	}); err != nil {
		return err
	}
	now := time.Now()
	for _, s := range SystemRoles() {
		_, err := r.col.UpdateOne(ctx,
			bson.M{"slug": s.Slug},
			bson.M{
				"$set": bson.M{
					"name":        s.Name,
					"description": s.Description,
					"permissions": s.Permissions,
					"isSystem":    true,
					"updatedAt":   now,
				},
				"$setOnInsert": bson.M{"_id": uuid.New(), "createdAt": now},
			},
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListAssignable returns the roles a non-wildcard admin may assign (for the
// create-user dropdown). The authoritative check is server-side at user
// creation; this only populates the UI.
func (r *RoleRepository) ListAssignable(ctx context.Context) ([]entity.Role, error) {
	all, err := r.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]entity.Role, 0, len(all))
	for _, role := range all {
		if entity.IsAssignableTenantRole(role.Slug) {
			out = append(out, role)
		}
	}
	return out, nil
}
