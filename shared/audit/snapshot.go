package audit

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Snapshot turns a domain document into the flat map an audit entry stores. It
// goes through BSON deliberately: the trail then shows exactly the field names and
// shapes that are persisted, so a reviewer comparing the trail against the
// database is comparing like with like.
//
// A marshalling failure yields nil rather than an error — an unsnapshottable
// document must not block the user's actual work (audit is best-effort).
func Snapshot(v any) map[string]any {
	if v == nil {
		return nil
	}
	raw, err := bson.Marshal(v)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := bson.Unmarshal(raw, &out); err != nil {
		return nil

	}
	return out
}

// AfterDelete derives a delete's "after" state from its "before" snapshot: a
// shallow copy with nameField suffixed " [Deleted]" and deletedAt/updatedAt
// stamped — the same shape SoftDelete's server-side $concat pipeline update
// produces. Computed here rather than re-read from the repository because a
// second FindByID after the soft delete would itself be filtered out by
// deletedAt:nil and find nothing. nil in ⇒ nil out (no before snapshot means
// audit was never going to record this action anyway).
func AfterDelete(before map[string]any, nameField string, at time.Time) map[string]any {
	if before == nil {
		return nil
	}
	after := make(map[string]any, len(before))
	for k, v := range before {
		after[k] = v
	}
	if name, ok := after[nameField].(string); ok {
		after[nameField] = name + " [Deleted]"
	}
	after["deletedAt"] = at
	after["updatedAt"] = at
	return after
}
