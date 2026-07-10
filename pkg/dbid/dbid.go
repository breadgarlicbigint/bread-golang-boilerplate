// Package dbid provides a configurable MongoDB _id strategy.
// Two strategies are supported:
//
//   - "uuid"     — uuid.UUID stored as BSON Binary subtype 4 (RFC 4122).
//     Globally unique without coordination; human-readable in logs.
//   - "objectid" — primitive.ObjectID (12-byte BSON type).
//     Sortable by insertion time; smaller on disk; MongoDB default.
//
// # Selecting a strategy
//
// Set DB_ID_TYPE in your .env file:
//
//	DB_ID_TYPE=uuid      # default
//	DB_ID_TYPE=objectid
//
// The selected strategy is wired in shared/database/mongo.go via
// NewMongoDB, which calls dbid.NewRegistry(cfg.DBIDType).
//
// # Switching strategies
//
// 1. Change DB_ID_TYPE in .env.
// 2. Update entity _id field types (uuid.UUID ↔ primitive.ObjectID).
// 3. Run a migration to convert existing documents.
// See docs/id-migration.md for the migration playbook.
//
// # Using in code
//
// Use the helper functions so call-sites don't need to import
// both uuid and primitive packages:
//
//	import "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/dbid"
//
//	// Generate a new ID (works regardless of active strategy)
//	id := dbid.NewUUID()      // → uuid.UUID (when strategy = uuid)
//	id := dbid.NewObjectID()  // → primitive.ObjectID (when strategy = objectid)
//
//	// Parse a string ID from a URL param or JWT claim
//	uid, err := dbid.ParseUUID(c.Param("id"))
//	oid, err := dbid.ParseObjectID(c.Param("id"))
package dbid

import (
	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/uuidbson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Strategy names — match DB_ID_TYPE config value.
const (
	StrategyUUID     = "uuid"
	StrategyObjectID = "objectid"
)

// NewRegistry returns the appropriate BSON codec registry for the strategy.
// Pass the returned registry to options.Client().SetRegistry(reg).
func NewRegistry(strategy string) *bsoncodec.Registry {
	switch strategy {
	case StrategyUUID:
		return uuidbson.NewRegistry()
	default: // StrategyObjectID or empty → MongoDB default
		rb := bsoncodec.NewRegistryBuilder()
		bsoncodec.DefaultValueEncoders{}.RegisterDefaultEncoders(rb)
		bsoncodec.DefaultValueDecoders{}.RegisterDefaultDecoders(rb)
		return rb.Build()
	}
}

// ── UUID helpers ──────────────────────────────────────────────────────────────

// NewUUID generates a new random UUID v4.
// Use when DB_ID_TYPE=uuid.
func NewUUID() uuid.UUID { return uuid.New() }

// ParseUUID parses a UUID string from a URL param, JWT claim, etc.
// Use instead of uuid.Parse to keep call-sites readable.
func ParseUUID(s string) (uuid.UUID, error) { return uuid.Parse(s) }

// MustParseUUID parses a UUID string and panics on error. Tests only.
func MustParseUUID(s string) uuid.UUID { return uuid.MustParse(s) }

// ── ObjectID helpers ──────────────────────────────────────────────────────────

// NewObjectID generates a new MongoDB ObjectID.
// Use when DB_ID_TYPE=objectid.
func NewObjectID() primitive.ObjectID { return primitive.NewObjectID() }

// ParseObjectID parses a 24-character hex ObjectID string.
func ParseObjectID(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}

// MustParseObjectID parses an ObjectID string and panics on error. Tests only.
func MustParseObjectID(s string) primitive.ObjectID {
	return func() primitive.ObjectID { oid, _ := primitive.ObjectIDFromHex(s); return oid }()
}
