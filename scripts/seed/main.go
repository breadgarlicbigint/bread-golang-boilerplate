// Package main seeds the database with default roles, users, feature flags,
// and app version policies. Each module's seed data lives in its own file
// (role.go, user.go, featureflag.go, appversion.go) — this file only wires
// the DB connection and calls them in order.
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
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
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
	adminUser := seedUser(ctx, db, h, "admin@example.com", "admin", "System", "Admin", "Admin@1234", adminRole.ID, true)
	testUser := seedUser(ctx, db, h, "user@example.com", "testuser", "Test", "User", "User@1234", userRole.ID, true)
	_, _ = adminUser, testUser

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
