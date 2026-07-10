// Package main seeds the database with default roles, users, feature flags,
// notification preferences, and app version policies.
//
// Run inside Docker network:
//
//	make seed
//
// Run locally against localhost:27017:
//
//	make seed-local
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	roleEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	userEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Allow MONGO_URI override via environment (used by make seed-local)
	applyEnvOverrides()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	db, err := database.NewMongoDB(cfg.Mongo)
	if err != nil {
		log.Fatal("mongo: ", err)
	}
	defer db.Disconnect(context.Background()) //nolint:errcheck

	h := hash.New(12)

	fmt.Println("▶ Seeding roles...")
	adminRole, userRole := seedRoles(ctx, db)

	fmt.Println("▶ Seeding users...")
	adminUser := seedUser(ctx, db, h, "admin@example.com",  "admin",    "System",  "Admin", "Admin@1234",  adminRole.ID, true)
	testUser  := seedUser(ctx, db, h, "user@example.com",   "testuser", "Test",    "User",  "User@1234",   userRole.ID,  true)
	_, _       = adminUser, testUser

	fmt.Println("▶ Seeding feature flags...")
	seedFeatureFlags(ctx, db)

	fmt.Println("▶ Seeding app versions...")
	seedAppVersions(ctx, db)

	fmt.Println("✅  Seed complete")
	fmt.Println()
	fmt.Println("  Credentials:")
	fmt.Println("    admin@example.com  /  Admin@1234  (role: admin)")
	fmt.Println("    user@example.com   /  User@1234   (role: user)")
}

// applyEnvOverrides lets callers override individual settings via env vars
// without needing a full .env file (used by make seed-local / CI).
func applyEnvOverrides() {
	if uri := os.Getenv("MONGO_URI"); uri != "" {
		os.Setenv("MONGO_URI", uri) // already set — viper will pick it up
	}
	if db := os.Getenv("MONGO_DB_NAME"); db == "" {
		os.Setenv("MONGO_DB_NAME", "bread_boilerplate")
	}
}

// ── Roles ─────────────────────────────────────────────────────────────────────

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

// ── Users ─────────────────────────────────────────────────────────────────────

func seedUser(
	ctx context.Context, db *database.MongoDB, h *hash.Hasher,
	email, username, firstName, lastName, password string,
	roleID uuid.UUID, emailVerified bool,
) *userEntity.User {
	col := db.Collection("users")

	pw, err := h.Hash(password)
	if err != nil {
		log.Printf("  ⚠ hash %s: %v", email, err)
		return nil
	}

	now := time.Now()
	u := &userEntity.User{
		ID:            uuid.New(),
		Email:         email,
		Username:      username,
		PasswordHash:  pw,
		FirstName:     firstName,
		LastName:      lastName,
		Status:        userEntity.UserStatusActive,
		RoleID:        roleID,
		EmailVerified: emailVerified,
		NotifSettings: userEntity.DefaultNotifSettings(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := upsert(ctx, col, bson.M{"email": email}, u); err != nil {
		log.Printf("  ⚠ user %s: %v", email, err)
	} else {
		fmt.Printf("  user: %-30s (password: %s) ✅\n", email, password)
	}
	return u
}

// ── Feature flags ─────────────────────────────────────────────────────────────

func seedFeatureFlags(ctx context.Context, db *database.MongoDB) {
	col := db.Collection("feature_flags")
	now := time.Now()

	flags := []bson.M{
		{"key": "new_dashboard",        "name": "New Dashboard",        "enabled": false, "rolloutPct": 0},
		{"key": "beta_api",             "name": "Beta API",             "enabled": true,  "rolloutPct": 100},
		{"key": "email_notifications",  "name": "Email Notifications",  "enabled": true,  "rolloutPct": 100},
		{"key": "push_notifications",   "name": "Push Notifications",   "enabled": false, "rolloutPct": 0},
		{"key": "passkey_login",        "name": "Passkey Login",        "enabled": true,  "rolloutPct": 100},
		{"key": "mobile_verification",  "name": "Mobile Verification",  "enabled": true,  "rolloutPct": 100},
		{"key": "whatsapp_otp",         "name": "WhatsApp OTP",         "enabled": false, "rolloutPct": 0},
		{"key": "multi_tenant",         "name": "Multi-Tenant Mode",    "enabled": false, "rolloutPct": 0},
	}

	for _, f := range flags {
		update := bson.M{
			// $setOnInsert: only on first insert — immutable fields
			"$setOnInsert": bson.M{
				"_id":       uuid.New(),
				"key":       f["key"],
				"createdAt": now,
			},
			// $set: applied on every run — keeps seed data up to date
			"$set": bson.M{
				"name":        f["name"],
				"enabled":     f["enabled"],
				"rolloutPct":  f["rolloutPct"],
				"updatedAt":   now,
			},
		}
		if _, err := col.UpdateOne(ctx, bson.M{"key": f["key"]}, update, options.Update().SetUpsert(true)); err != nil {
			log.Printf("  ⚠ flag %s: %v", f["key"], err)
		} else {
			fmt.Printf("  flag: %-25s enabled=%-5v ✅\n", f["key"], f["enabled"])
		}
	}
}

// ── App versions ──────────────────────────────────────────────────────────────

func seedAppVersions(ctx context.Context, db *database.MongoDB) {
	col := db.Collection("app_versions")
	now := time.Now()

	platforms := []struct {
		platform string
		current  string
		min      string
		storeURL string
	}{
		{"ios",     "1.0.0", "1.0.0", "https://apps.apple.com/app/yourapp"},
		{"android", "1.0.0", "1.0.0", "https://play.google.com/store/apps/details?id=com.yourapp"},
		{"web",     "1.0.0", "1.0.0", ""},
	}

	for _, p := range platforms {
		update := bson.M{
			// $setOnInsert: immutable — only written on first insert
			"$setOnInsert": bson.M{
				"_id":       uuid.New(),
				"platform":  p.platform,
				"createdAt": now,
			},
			// $set: mutable — updated on every seed run
			"$set": bson.M{
				"currentVersion": p.current,
				"minVersion":     p.min,
				"forceUpdate":    false,
				"storeUrl":       p.storeURL,
				"updatedAt":      now,
			},
		}
		if _, err := col.UpdateOne(ctx, bson.M{"platform": p.platform}, update, options.Update().SetUpsert(true)); err != nil {
			log.Printf("  ⚠ version %s: %v", p.platform, err)
		} else {
			fmt.Printf("  version: %-10s current=%s min=%s ✅\n", p.platform, p.current, p.min)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func upsert(ctx context.Context, col *mongo.Collection, filter bson.M, doc interface{}) error {
	update := bson.M{
		"$setOnInsert": doc,
	}
	_, err := col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}
