package main

import (
	"context"
	"fmt"
	"log"
	"time"

	roleEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// seedRoles creates the system roles (admin, user, member) and returns the
// admin/user roles so seedUser can assign them.
func seedRoles(ctx context.Context, db *database.MongoDB) (*roleEntity.Role, *roleEntity.Role) {
	col := db.Collection("roles")
	now := time.Now()

	roles := []roleEntity.Role{
		{
			ID:          uuid.New(),
			Name:        "Administrator",
			Slug:        roleEntity.RoleAdmin,
			Description: "Full system access",
			Permissions: []roleEntity.Permission{roleEntity.PermAll},
			IsSystem:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			Name:        "User",
			Slug:        roleEntity.RoleUser,
			Description: "Standard user access",
			Permissions: []roleEntity.Permission{roleEntity.PermUserRead},
			IsSystem:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          uuid.New(),
			Name:        "Member",
			Slug:        roleEntity.RoleMember,
			Description: "Limited member access",
			Permissions: []roleEntity.Permission{},
			IsSystem:    false,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	var adminRole, userRole *roleEntity.Role
	for i := range roles {
		r := &roles[i]
		if err := upsert(ctx, col, bson.M{"slug": r.Slug}, r); err != nil {
			log.Printf("  ⚠ role %s: %v", r.Slug, err)
		} else {
			fmt.Printf("  role: %-10s ✅\n", r.Slug)
		}
		switch r.Slug {
		case roleEntity.RoleAdmin:
			adminRole = r
		case roleEntity.RoleUser:
			userRole = r
		}
	}
	return adminRole, userRole
}
