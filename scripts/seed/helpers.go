package main

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// upsert inserts doc if no document matches filter yet; a no-op otherwise.
// Shared by every per-module seed file below.
func upsert(ctx context.Context, col *mongo.Collection, filter bson.M, doc any) error {
	update := bson.M{
		"$setOnInsert": doc,
	}
	_, err := col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}
