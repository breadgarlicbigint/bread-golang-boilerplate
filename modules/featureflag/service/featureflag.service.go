package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const flagCollection = "feature_flags"
const flagCachePrefix = "flag:"
const flagCacheTTL = 5 * time.Minute

// FeatureFlag is stored in MongoDB and cached in Redis.
type FeatureFlag struct {
	ID          uuid.UUID `bson:"_id,omitempty" json:"id"`
	Key         string             `bson:"key"           json:"key"`      // e.g. "new_dashboard"
	Name        string             `bson:"name"          json:"name"`
	Description string             `bson:"description"   json:"description"`
	Enabled     bool               `bson:"enabled"       json:"enabled"`
	RolloutPct  int                `bson:"rolloutPct"    json:"rolloutPct"`  // 0-100, % of users
	AllowedRoles []string          `bson:"allowedRoles"  json:"allowedRoles"` // empty = all
	CreatedAt   time.Time          `bson:"createdAt"     json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"     json:"updatedAt"`
}

type FeatureFlagService struct {
	col *mongo.Collection
	rdb *redis.Client
}

func New(db *database.MongoDB, rdb *redis.Client) *FeatureFlagService {
	return &FeatureFlagService{col: db.Collection(flagCollection), rdb: rdb}
}

// IsEnabled returns true if the flag exists and is enabled.
// It checks Redis cache first, falling back to MongoDB.
func (s *FeatureFlagService) IsEnabled(ctx context.Context, key string) (bool, error) {
	cacheKey := flagCachePrefix + key

	// Cache hit
	val, err := s.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		return val == "1", nil
	}

	var flag FeatureFlag
	err = s.col.FindOne(ctx, bson.M{"key": key}).Decode(&flag)
	if err == mongo.ErrNoDocuments {
		_ = s.rdb.Set(ctx, cacheKey, "0", flagCacheTTL)
		return false, nil
	}
	if err != nil {
		return false, err
	}

	cacheVal := "0"
	if flag.Enabled {
		cacheVal = "1"
	}
	_ = s.rdb.Set(ctx, cacheKey, cacheVal, flagCacheTTL)
	return flag.Enabled, nil
}

// IsEnabledForRole checks if a flag is active for the given role.
func (s *FeatureFlagService) IsEnabledForRole(ctx context.Context, key, role string) (bool, error) {
	var flag FeatureFlag
	if err := s.col.FindOne(ctx, bson.M{"key": key}).Decode(&flag); err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	if !flag.Enabled {
		return false, nil
	}
	if len(flag.AllowedRoles) == 0 {
		return true, nil
	}
	for _, r := range flag.AllowedRoles {
		if r == role {
			return true, nil
		}
	}
	return false, nil
}

// Upsert creates or updates a feature flag.
func (s *FeatureFlagService) Upsert(ctx context.Context, key, name, description string, enabled bool, rolloutPct int, allowedRoles []string) (*FeatureFlag, error) {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"key": key, "name": name, "description": description,
			"enabled": enabled, "rolloutPct": rolloutPct,
			"allowedRoles": allowedRoles, "updatedAt": now,
		},
		"$setOnInsert": bson.M{"_id": uuid.New(), "createdAt": now},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var flag FeatureFlag
	if err := s.col.FindOneAndUpdate(ctx, bson.M{"key": key}, update, opts).Decode(&flag); err != nil {
		return nil, err
	}

	// Bust cache
	_ = s.rdb.Del(ctx, fmt.Sprintf("%s%s", flagCachePrefix, key))
	return &flag, nil
}

// List returns all feature flags.
func (s *FeatureFlagService) List(ctx context.Context) ([]FeatureFlag, error) {
	cur, err := s.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var flags []FeatureFlag
	return flags, cur.All(ctx, &flags)
}

// FeatureFlagMiddleware returns a Gin handler that blocks requests when a
// flag is disabled. Import from middleware package to use it.
func (s *FeatureFlagService) IsEnabledFunc(key string) func(ctx context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		return s.IsEnabled(ctx, key)
	}
}
