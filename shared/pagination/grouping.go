package pagination

import (
	"net/http"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
)

// Server-side grouping, multi-level and ROW-PAGED.
//
// `groupBy` is a repeatable query param; its order is the nesting order. Paging
// stays over raw records exactly as when ungrouped — `page`/`perPage` window the
// row set and `total` is the record count — so a group spans page boundaries and
// the client repeats its header on the next page.
//
// This replaced a two-phase design in which the pager counted GROUPS.
// That made "12 records at perPage 10" collapse to a single page as soon as the
// rows fell into one group, which is what read to the user as pagination being
// broken.
//
// A level is always a client-facing key resolved through a per-module whitelist,
// never a raw Mongo path: a user-supplied string must not reach a `$group._id`
// or a `$sort`.

// ErrInvalidGroupBy is returned when the requested grouping is not something the
// endpoint exposes. The message deliberately names no identifier (CLAUDE.md
// the no-identifiers-in-prose rule) — the offending value is a wire identifier, not prose.
var ErrInvalidGroupBy = apperrors.NewI18n(
	http.StatusBadRequest,
	"INVALID_GROUP_BY",
	"pagination.invalidGroupBy",
	"This list cannot be grouped by the selected column",
)

// GroupKind tells the decoder how a level's values are stored, so a raw BSON
// value can be turned back into something meaningful on the wire.
type GroupKind string

const (
	GroupString GroupKind = "string"
	GroupBool   GroupKind = "bool"
	GroupUUID   GroupKind = "uuid"
)

// MaxGroupDepth caps how deeply a list may be grouped. Beyond three the header
// rows outnumber the data on any realistic screen.
const MaxGroupDepth = 3

// GroupField describes one groupable column: where it lives in the document and
// what type its values are.
type GroupField struct {
	// Path is the bson dotted path (e.g. "fitment.carCompanyId"). Module-owned
	// and never client-supplied.
	Path string
	Kind GroupKind
}

// GroupFields is a module's whitelist of client-facing group key → field.
type GroupFields map[string]GroupField

// Resolve looks up one client-supplied group key.
func (f GroupFields) Resolve(key string) (GroupField, error) {
	gf, ok := f[key]
	if !ok {
		return GroupField{}, ErrInvalidGroupBy
	}
	return gf, nil
}

// Levels validates the whole requested grouping and returns the fields in
// nesting order. An ungrouped query yields nil with no error, so callers can
// branch on the length.
func (f GroupFields) Levels(q Query) ([]GroupField, error) {
	if !q.IsGrouped() {
		return nil, nil
	}
	if len(q.GroupBy) > MaxGroupDepth {
		return nil, ErrInvalidGroupBy
	}
	out := make([]GroupField, 0, len(q.GroupBy))
	seen := make(map[string]struct{}, len(q.GroupBy))
	for _, name := range q.GroupBy {
		gf, err := f.Resolve(name)
		if err != nil {
			return nil, err
		}
		// Grouping by the same column twice yields exactly one child group per
		// parent — meaningless, and it reads as a rendering bug rather than a
		// bad request.
		if _, dup := seen[name]; dup {
			return nil, ErrInvalidGroupBy
		}
		seen[name] = struct{}{}
		out = append(out, gf)
	}
	return out, nil
}

// GroupedRow is one row plus the group it belongs to. Wrapping rather than
// annotating the entity keeps the document's own JSON shape untouched, so no
// consumer of a row is affected by grouping being switched on.
type GroupedRow struct {
	GroupPath []any `json:"groupPath"`
	Row       any   `json:"row"`
}

// GroupCount is the FULL size of one group or sub-group — not its size within
// the current page, so a header can read "(12 items)" while showing 10.
type GroupCount struct {
	Path  []any `json:"path"`
	Count int64 `json:"count"`
}

// DecodeGroupKey normalises a decoded group value into something meaningful on
// the wire.
//
// This has to exist because UUIDs are stored as BSON Binary subtype 4
// (pkg/uuidbson) — one decoded into an `any` arrives as a primitive.Binary,
// which marshals as `{"Subtype":4,"Data":"…"}` and is useless to a client that
// has to match it against an id it already knows.
//
// It takes an `any` (not a bson.RawValue) deliberately: the application's BSON
// registry (pkg/uuidbson.NewRegistry) registers the default encoders/decoders
// but not bsoncodec's primitive codecs, so neither bson.Raw nor bson.RawValue
// has a decoder and a repository must read documents through bson.M.
//
// A missing or null field yields nil — the "no value" group. A value of an
// unexpected type degrades to that same nil rather than panicking.
func DecodeGroupKey(v any, kind GroupKind) any {
	if v == nil {
		return nil
	}
	switch kind {
	case GroupUUID:
		switch t := v.(type) {
		case primitive.Binary:
			id, err := uuid.FromBytes(t.Data)
			if err != nil {
				return nil
			}
			return id.String()
		case uuid.UUID:
			return t.String()
		case string:
			// Tolerated for the same reason the UUID codec accepts strings:
			// hand-inspected or migrated data may store the id as text.
			return t
		}
		return nil
	case GroupBool:
		b, ok := v.(bool)
		if !ok {
			return nil
		}
		return b
	default:
		s, ok := v.(string)
		if !ok {
			return nil
		}
		return s
	}
}

// GroupCountOf reads a `$sum` result, which comes back int32 while counts are
// small and int64 once they grow.
func GroupCountOf(v any) int64 {
	switch n := v.(type) {
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

// IsGrouped reports whether the request asked for grouping.
func (q *Query) IsGrouped() bool { return len(q.GroupBy) > 0 }
