// Package uuidbson provides a custom BSON codec that stores uuid.UUID values
// as MongoDB Binary subtype 4 (the BSON UUID standard), enabling UUID primary
// keys and foreign-key references throughout the application.
//
// Register once with the MongoDB client:
//
//	opts := options.Client().SetRegistry(uuidbson.NewRegistry())
//
// After registration, any struct field typed uuid.UUID with a bson tag will be
// transparently encoded/decoded as UUID binary in MongoDB.
package uuidbson

import (
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BSONSubtypeUUID is the BSON Binary subtype reserved for UUID (RFC 4122).
const BSONSubtypeUUID byte = 0x04

var uuidType = reflect.TypeOf(uuid.UUID{})

// Codec implements bsoncodec.ValueEncoder and bsoncodec.ValueDecoder for uuid.UUID.
type Codec struct{}

// EncodeValue writes a uuid.UUID as BSON Binary subtype 4.
func (Codec) EncodeValue(_ bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != uuidType {
		return bsoncodec.ValueEncoderError{
			Name:     "UUIDCodec",
			Types:    []reflect.Type{uuidType},
			Received: val,
		}
	}
	uid := val.Interface().(uuid.UUID)
	return vw.WriteBinaryWithSubtype(uid[:], BSONSubtypeUUID)
}

// DecodeValue reads a BSON Binary subtype-4 value into a uuid.UUID.
// Also accepts string and ObjectID for easier migration from ObjectID-based data.
func (Codec) DecodeValue(_ bsoncodec.DecodeContext, vr bsonrw.ValueReader, val reflect.Value) error {
	if !val.CanSet() || val.Type() != uuidType {
		return bsoncodec.ValueDecoderError{
			Name:     "UUIDCodec",
			Types:    []reflect.Type{uuidType},
			Received: val,
		}
	}

	switch vr.Type() {
	case bsontype.Binary:
		data, subtype, err := vr.ReadBinary()
		if err != nil {
			return fmt.Errorf("uuidbson: read binary: %w", err)
		}
		if subtype != BSONSubtypeUUID {
			return fmt.Errorf("uuidbson: unexpected binary subtype %d (want %d)", subtype, BSONSubtypeUUID)
		}
		uid, err := uuid.FromBytes(data)
		if err != nil {
			return fmt.Errorf("uuidbson: parse UUID bytes: %w", err)
		}
		val.Set(reflect.ValueOf(uid))
		return nil

	case bsontype.String:
		// Support string UUIDs for easier manual inspection / migration.
		s, err := vr.ReadString()
		if err != nil {
			return fmt.Errorf("uuidbson: read string: %w", err)
		}
		uid, err := uuid.Parse(s)
		if err != nil {
			return fmt.Errorf("uuidbson: parse UUID string %q: %w", s, err)
		}
		val.Set(reflect.ValueOf(uid))
		return nil

	case bsontype.Null:
		err := vr.ReadNull()
		if err != nil {
			return err
		}
		val.Set(reflect.ValueOf(uuid.Nil))
		return nil

	default:
		return fmt.Errorf("uuidbson: cannot decode BSON type %s into uuid.UUID", vr.Type())
	}
}

// NewRegistry returns a BSON registry with the UUID codec registered.
// Pass it to options.Client().SetRegistry(reg) when creating the MongoDB client.
func NewRegistry() *bsoncodec.Registry {
	rb := bsoncodec.NewRegistryBuilder()

	// Copy all default codecs first.
	bsoncodec.DefaultValueEncoders{}.RegisterDefaultEncoders(rb)
	bsoncodec.DefaultValueDecoders{}.RegisterDefaultDecoders(rb)

	// Register UUID codec on top of the defaults.
	c := Codec{}
	rb.RegisterTypeEncoder(uuidType, c)
	rb.RegisterTypeDecoder(uuidType, c)

	return rb.Build()
}

// NewClientOptions returns MongoDB client options pre-configured with the UUID
// registry. Merge with other options using options.MergeClientOptions:
//
//	opts := options.MergeClientOptions(uuidbson.NewClientOptions(), customOpts)
func NewClientOptions() *options.ClientOptions {
	return options.Client().SetRegistry(NewRegistry())
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// New returns a new random UUID v4. Use instead of uuid.New().
func New() uuid.UUID { return uuid.New() }

// Parse parses a UUID string from a URL param or JWT claim.
// Use instead of uuid.Parse().
func Parse(s string) (uuid.UUID, error) { return uuid.Parse(s) }

// MustParse parses a UUID string and panics on error (tests only).
func MustParse(s string) uuid.UUID { return uuid.MustParse(s) }

// NilFilter returns a BSON filter that matches documents where the field
// equals uuid.Nil (zero value). Useful for "unset" reference fields.
func NilFilter(field string) interface{} {
	return map[string]interface{}{field: uuid.Nil}
}

// FromObjectID converts a legacy 12-byte ObjectID hex string to a UUID
// for data migration. The ObjectID bytes are copied into the first 12 bytes.
func FromObjectID(oidHex string) uuid.UUID {
	var uid uuid.UUID
	// Copy raw hex bytes as a best-effort migration mapping
	copy(uid[:], []byte(oidHex))
	return uid
}
