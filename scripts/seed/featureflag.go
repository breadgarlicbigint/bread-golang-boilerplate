package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// seedFeatureFlags creates/updates the default feature flag set.
func seedFeatureFlags(ctx context.Context, db *database.MongoDB) {
	col := db.Collection("feature_flags")
	now := time.Now()

	flags := []bson.M{
		{"key": "new_dashboard", "name": "New Dashboard", "enabled": false, "rolloutPct": 0},
		{"key": "beta_api", "name": "Beta API", "enabled": true, "rolloutPct": 100},
		{"key": "email_notifications", "name": "Email Notifications", "enabled": true, "rolloutPct": 100},
		{"key": "push_notifications", "name": "Push Notifications", "enabled": false, "rolloutPct": 0},
		{"key": "passkey_login", "name": "Passkey Login", "enabled": true, "rolloutPct": 100},
		{"key": "mobile_verification", "name": "Mobile Verification", "enabled": true, "rolloutPct": 100},
		{"key": "whatsapp_otp", "name": "WhatsApp OTP", "enabled": false, "rolloutPct": 0},
		{"key": "multi_tenant", "name": "Multi-Tenant Mode", "enabled": false, "rolloutPct": 0},
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
				"name":       f["name"],
				"enabled":    f["enabled"],
				"rolloutPct": f["rolloutPct"],
				"updatedAt":  now,
			},
		}
		if _, err := col.UpdateOne(ctx, bson.M{"key": f["key"]}, update, options.Update().SetUpsert(true)); err != nil {
			log.Printf("  ⚠ flag %s: %v", f["key"], err)
		} else {
			fmt.Printf("  flag: %-25s enabled=%-5v ✅\n", f["key"], f["enabled"])
		}
	}
}
