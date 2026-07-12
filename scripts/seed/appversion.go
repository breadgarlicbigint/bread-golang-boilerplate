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

// seedAppVersions creates/updates the default version policy per platform.
func seedAppVersions(ctx context.Context, db *database.MongoDB) {
	col := db.Collection("app_versions")
	now := time.Now()

	platforms := []struct {
		platform string
		current  string
		min      string
		storeURL string
	}{
		{"ios", "1.0.0", "1.0.0", "https://apps.apple.com/app/yourapp"},
		{"android", "1.0.0", "1.0.0", "https://play.google.com/store/apps/details?id=com.yourapp"},
		{"web", "1.0.0", "1.0.0", ""},
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
