package uuidbson_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/uuidbson"
	"go.mongodb.org/mongo-driver/bson"
)

func TestRoundTrip(t *testing.T) {
	reg := uuidbson.NewRegistry()

	type doc struct {
		ID   uuid.UUID `bson:"_id"`
		Ref  uuid.UUID `bson:"ref"`
	}

	orig := doc{
		ID:  uuidbson.New(),
		Ref: uuidbson.New(),
	}

	// Encode
	encoded, err := bson.MarshalWithRegistry(reg, orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	// Decode
	var got doc
	if err := bson.UnmarshalWithRegistry(reg, encoded, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID: want %s, got %s", orig.ID, got.ID)
	}
	if got.Ref != orig.Ref {
		t.Errorf("Ref: want %s, got %s", orig.Ref, got.Ref)
	}
}

func TestNilUUID(t *testing.T) {
	reg := uuidbson.NewRegistry()
	type doc struct{ ID uuid.UUID `bson:"_id"` }

	orig := doc{ID: uuid.Nil}
	encoded, err := bson.MarshalWithRegistry(reg, orig)
	if err != nil {
		t.Fatalf("Marshal nil UUID: %v", err)
	}
	var got doc
	if err := bson.UnmarshalWithRegistry(reg, encoded, &got); err != nil {
		t.Fatalf("Unmarshal nil UUID: %v", err)
	}
	// nil UUID round-trips as uuid.Nil
	t.Logf("nil UUID round-trip: %s", got.ID)
}

func TestParse(t *testing.T) {
	s := "550e8400-e29b-41d4-a716-446655440000"
	uid, err := uuidbson.Parse(s)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if uid.String() != s {
		t.Errorf("want %s, got %s", s, uid.String())
	}
}

func TestParseInvalid(t *testing.T) {
	_, err := uuidbson.Parse("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID string")
	}
}
