package service

import (
	"context"
	"fmt"
	"net/http"
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

// maxPrefixAttempts bounds the lookup-prefix re-roll in Create. See Create.
const maxPrefixAttempts = 32

// ErrKeyPrefixExhausted: every lookup prefix Create tried belongs to an active
// key. With a short key prefix there are only so many 8-char lookup prefixes, so
// this means a very large number of active keys — revoke unused ones.
var ErrKeyPrefixExhausted = errors.NewI18n(http.StatusInternalServerError, "API_KEY_PREFIX_EXHAUSTED", "http.500", "No free API key prefix; revoke unused keys")

type APIKeyService struct {
	col    *mongo.Collection
	hasher *hash.Hasher
	prefix string
	// prefixTaken reports whether an ACTIVE key already uses this 8-char lookup
	// prefix. A field so tests can exercise the re-roll without a live Mongo;
	// New wires it to the CountDocuments query.
	prefixTaken func(ctx context.Context, prefix8 string) (bool, error)
}

func New(db *database.MongoDB, hasher *hash.Hasher, prefix string) *APIKeyService {
	s := &APIKeyService{col: db.Collection(collection), hasher: hasher, prefix: prefix}
	s.prefixTaken = s.prefixTakenInDB
	return s
}

func (s *APIKeyService) prefixTakenInDB(ctx context.Context, prefix8 string) (bool, error) {
	n, err := s.col.CountDocuments(ctx, bson.M{"keyPrefix": prefix8, "isActive": true})
	return n > 0, err
}

// freePrefix generates a raw key whose 8-char lookup prefix is free among ACTIVE
// keys, re-rolling up to maxPrefixAttempts times. See Create for why.
func (s *APIKeyService) freePrefix(ctx context.Context) (string, error) {
	for attempt := 0; attempt < maxPrefixAttempts; attempt++ {
		raw, err := hash.RandomHex(32) // 64-char hex key
		if err != nil {
			return "", err
		}
		candidate := fmt.Sprintf("%s_%s", s.prefix, raw)
		taken, err := s.prefixTaken(ctx, candidate[:8])
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", ErrKeyPrefixExhausted
}

// Create generates a new key, hashes it, persists it, and returns the raw value once.
//
// The key is looked up by its first 8 characters (Validate), and a lookup takes
// the FIRST active match. With a short default prefix only a couple of those 8
// chars are random, so two active keys collide roughly 1 in 256 — and the second
// is then rejected on every request. Create therefore re-rolls until the lookup
// prefix is free among ACTIVE keys. A revoked key's prefix may be reused:
// Validate never matches an inactive row.
func (s *APIKeyService) Create(ctx context.Context, name, description string, keyType entity.APIKeyType,
	permissions []string, expiresAt *time.Time, createdBy uuid.UUID) (rawKey string, k *entity.APIKey, err error) {

	rawKey, err = s.freePrefix(ctx)
	if err != nil {
		return "", nil, err
	}

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
	if len(parts) != 2 || parts[0] != s.prefix || len(rawKey) < 8 {
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
