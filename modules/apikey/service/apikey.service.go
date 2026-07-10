package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/apikey/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

const collection = "api_keys"

type APIKeyService struct {
	col    *mongo.Collection
	hasher *hash.Hasher
	prefix string
}

func New(db *database.MongoDB, hasher *hash.Hasher, prefix string) *APIKeyService {
	return &APIKeyService{col: db.Collection(collection), hasher: hasher, prefix: prefix}
}

// Create generates a new key, hashes it, persists it, and returns the raw value once.
func (s *APIKeyService) Create(ctx context.Context, name, description string, keyType entity.APIKeyType,
	permissions []string, expiresAt *time.Time, createdBy uuid.UUID) (rawKey string, k *entity.APIKey, err error) {

	raw, err := hash.RandomHex(32) // 64-char hex key
	if err != nil {
		return "", nil, err
	}
	rawKey = fmt.Sprintf("%s_%s", s.prefix, raw)

	keyHash, err := s.hasher.Hash(rawKey)
	if err != nil {
		return "", nil, err
	}

	now := time.Now()
	k = &entity.APIKey{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		KeyHash:     keyHash,
		KeyPrefix:   rawKey[:8],
		Type:        keyType,
		IsActive:    true,
		Permissions: permissions,
		ExpiresAt:   expiresAt,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := s.col.InsertOne(ctx, k); err != nil {
		return "", nil, err
	}
	return rawKey, k, nil
}

// Validate implements middleware.APIKeyValidator.
// It looks up the key by its 8-char prefix then compares the full hash.
func (s *APIKeyService) Validate(ctx context.Context, rawKey string) (string, error) {
	parts := strings.SplitN(rawKey, "_", 2)
	if len(parts) != 2 || parts[0] != s.prefix {
		return "", errors.ErrAPIKeyInvalid
	}

	prefix := rawKey[:8]
	var k entity.APIKey
	if err := s.col.FindOne(ctx, bson.M{"keyPrefix": prefix, "isActive": true}).Decode(&k); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.ErrAPIKeyInvalid
		}
		return "", err
	}

	if k.IsExpired() {
		return "", errors.ErrAPIKeyExpired
	}

	if !s.hasher.Compare(rawKey, k.KeyHash) {
		return "", errors.ErrAPIKeyInvalid
	}

	// Update lastUsedAt asynchronously
	go func() {
		now := time.Now()
		_, _ = s.col.UpdateOne(context.Background(), bson.M{"_id": k.ID},
			bson.M{"$set": bson.M{"lastUsedAt": now}})
	}()

	return k.ID.String(), nil
}

// Revoke deactivates an API key.
func (s *APIKeyService) Revoke(ctx context.Context, keyID uuid.UUID) error {
	_, err := s.col.UpdateOne(ctx, bson.M{"_id": keyID},
		bson.M{"$set": bson.M{"isActive": false, "updatedAt": time.Now()}})
	return err
}

// List returns all keys for a creator (hashes excluded).
func (s *APIKeyService) List(ctx context.Context, createdBy uuid.UUID) ([]entity.APIKey, error) {
	cur, err := s.col.Find(ctx, bson.M{"createdBy": createdBy})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var keys []entity.APIKey
	return keys, cur.All(ctx, &keys)
}
