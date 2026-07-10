// Package main creates all MongoDB indexes for every collection.
//
// Run inside Docker network:
//
//	make migrate-indexes
//
// Run locally against localhost:27017:
//
//	make migrate-indexes-local
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type indexSpec struct {
	collection string
	models     []mongo.IndexModel
}

func main() {
	// Allow MONGO_URI override via environment
	if db := os.Getenv("MONGO_DB_NAME"); db == "" {
		os.Setenv("MONGO_DB_NAME", "bread_boilerplate")
	}

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

	specs := allIndexSpecs()
	total := 0

	for _, spec := range specs {
		col := db.Collection(spec.collection)
		names, err := col.Indexes().CreateMany(ctx, spec.models)
		if err != nil {
			log.Printf("  ⚠  %-30s %v", spec.collection, err)
			continue
		}
		for _, n := range names {
			fmt.Printf("  ✅  %-30s %s\n", spec.collection, n)
			total++
		}
	}

	fmt.Printf("\n▶ Migration complete — %d indexes created/verified\n", total)
}

func allIndexSpecs() []indexSpec {
	sparse    := func(name string) *options.IndexOptions { return options.Index().SetSparse(true).SetName(name) }
	unique    := func(name string) *options.IndexOptions { return options.Index().SetUnique(true).SetName(name) }
	sparseUnq := func(name string) *options.IndexOptions { return options.Index().SetSparse(true).SetUnique(true).SetName(name) }
	plain     := func(name string) *options.IndexOptions { return options.Index().SetName(name) }
	ttl       := func(name string, secs int32) *options.IndexOptions { return options.Index().SetExpireAfterSeconds(secs).SetName(name) }

	return []indexSpec{
		// ── users ────────────────────────────────────────────────────────────
		{collection: "users", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "email",    Value: 1}}, Options: sparseUnq("idx_email_unique")},
			{Keys: bson.D{{Key: "username", Value: 1}}, Options: sparseUnq("idx_username_unique")},
			{Keys: bson.D{{Key: "googleId", Value: 1}}, Options: sparse("idx_google_id")},
			{Keys: bson.D{{Key: "appleId",  Value: 1}}, Options: sparse("idx_apple_id")},
			{Keys: bson.D{{Key: "status",   Value: 1}}, Options: plain("idx_status")},
			{Keys: bson.D{{Key: "roleId",   Value: 1}}, Options: plain("idx_role_id")},
			{Keys: bson.D{{Key: "deletedAt", Value: 1}}, Options: plain("idx_deleted_at")},
			{Keys: bson.D{{Key: "createdAt", Value: -1}}, Options: plain("idx_created_at_desc")},
		}},

		// ── roles ─────────────────────────────────────────────────────────────
		{collection: "roles", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "slug", Value: 1}}, Options: unique("idx_slug_unique")},
		}},

		// ── api_keys ──────────────────────────────────────────────────────────
		{collection: "api_keys", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "keyPrefix", Value: 1}}, Options: plain("idx_key_prefix")},
			{Keys: bson.D{{Key: "createdBy", Value: 1}}, Options: plain("idx_created_by")},
			{Keys: bson.D{{Key: "isActive",  Value: 1}}, Options: plain("idx_is_active")},
			{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: sparse("idx_expires_at")},
		}},

		// ── activity_logs ─────────────────────────────────────────────────────
		{collection: "activity_logs", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "actorId",       Value: 1}}, Options: plain("idx_actor_id")},
			{Keys: bson.D{{Key: "action",        Value: 1}}, Options: plain("idx_action")},
			{Keys: bson.D{{Key: "correlationId", Value: 1}}, Options: plain("idx_correlation_id")},
			{Keys: bson.D{{Key: "direction",     Value: 1}}, Options: plain("idx_direction")},
			{Keys: bson.D{{Key: "channel",       Value: 1}}, Options: sparse("idx_channel")},
			{Keys: bson.D{{Key: "tenantId",      Value: 1}}, Options: sparse("idx_tenant_id")},
			{Keys: bson.D{{Key: "createdAt",     Value: -1}}, Options: plain("idx_created_at")},
			// TTL: auto-delete after 90 days
			{Keys: bson.D{{Key: "createdAt", Value: 1}}, Options: ttl("idx_ttl_90d", 7776000)},
		}},

		// ── passkeys ──────────────────────────────────────────────────────────
		{collection: "passkeys", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId",            Value: 1}}, Options: plain("idx_user_id")},
			{Keys: bson.D{{Key: "credentialIdBase64", Value: 1}}, Options: unique("idx_cred_id_unique")},
			{Keys: bson.D{{Key: "tenantId",          Value: 1}}, Options: sparse("idx_tenant_id")},
		}},

		// ── feature_flags ─────────────────────────────────────────────────────
		{collection: "feature_flags", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "key", Value: 1}}, Options: unique("idx_key_unique")},
		}},

		// ── tenants ───────────────────────────────────────────────────────────
		{collection: "tenants", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "slug",   Value: 1}}, Options: unique("idx_slug_unique")},
			{Keys: bson.D{{Key: "domain", Value: 1}}, Options: sparse("idx_domain")},
			{Keys: bson.D{{Key: "ownerId", Value: 1}}, Options: plain("idx_owner_id")},
			{Keys: bson.D{{Key: "status", Value: 1}}, Options: plain("idx_status")},
		}},

		// ── tenant_members ────────────────────────────────────────────────────
		{collection: "tenant_members", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "userId", Value: 1}}, Options: unique("idx_tenant_user_unique")},
		}},

		// ── user_mobiles ──────────────────────────────────────────────────────
		{collection: "user_mobiles", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}}, Options: plain("idx_user_id")},
			{Keys: bson.D{{Key: "e164",   Value: 1}}, Options: sparseUnq("idx_e164_unique")},
			{Keys: bson.D{{Key: "isVerified", Value: 1}}, Options: plain("idx_verified")},
		}},

		// ── app_versions ──────────────────────────────────────────────────────
		{collection: "app_versions", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "platform", Value: 1}}, Options: unique("idx_platform_unique")},
		}},

		// ── notifications ─────────────────────────────────────────────────────
		{collection: "notifications", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "isRead", Value: 1}}, Options: plain("idx_user_read")},
			{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}, Options: plain("idx_user_created")},
			{Keys: bson.D{{Key: "type",   Value: 1}}, Options: plain("idx_type")},
			{Keys: bson.D{{Key: "tenantId", Value: 1}}, Options: sparse("idx_tenant_id")},
			// TTL: auto-delete after 90 days
			{Keys: bson.D{{Key: "createdAt", Value: 1}}, Options: ttl("idx_ttl_90d", 7776000)},
		}},

		// ── device_tokens ─────────────────────────────────────────────────────
		{collection: "device_tokens", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId",   Value: 1}}, Options: plain("idx_user_id")},
			{Keys: bson.D{{Key: "token",    Value: 1}}, Options: unique("idx_token_unique")},
			{Keys: bson.D{{Key: "isActive", Value: 1}}, Options: plain("idx_is_active")},
		}},

		// ── notification_preferences ──────────────────────────────────────────
		{collection: "notification_preferences", models: []mongo.IndexModel{
			{Keys: bson.D{{Key: "userId", Value: 1}}, Options: unique("idx_user_id_unique")},
		}},
	}
}
