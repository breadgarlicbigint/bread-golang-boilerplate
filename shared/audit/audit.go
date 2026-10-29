// Package audit defines the change-record contract every mutating service writes
// to: WHO changed WHAT, with the document BEFORE and AFTER. It lets an operator
// resolve a dispute ("who changed that?") and spot abuse patterns.
//
// This package holds only types + pure helpers, so any module can depend on it
// without pulling in a database. The writer lives in modules/activity.
package audit

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/google/uuid"
)

// Risk marks how closely a change should be watched. HIGH actions are the ones a
// dishonest or dangerous actor would use: they get their own filter and can feed
// a digest.
type Risk string

const (
	RiskNormal Risk = "NORMAL"
	RiskHigh   Risk = "HIGH"
)

// Entry is one audit record. Before/After hold the document state around the
// change; ChangedFields is the shallow diff, precomputed so the UI and any query
// can filter on it without re-diffing.
type Entry struct {
	ID             uuid.UUID      `bson:"_id"            json:"id"`
	TenantID       uuid.UUID      `bson:"tenantId"       json:"tenantId"`
	CorrelationID  string         `bson:"correlationId"  json:"correlationId,omitempty"`
	ActorID        string         `bson:"actorId"        json:"actorId"`
	ActorRole      string         `bson:"actorRole"      json:"actorRole,omitempty"`
	ClientPlatform string         `bson:"clientPlatform" json:"clientPlatform,omitempty"`
	IPAddress      string         `bson:"ipAddress"      json:"ipAddress,omitempty"`
	Action         string         `bson:"action"         json:"action"`
	EntityType     string         `bson:"entityType"     json:"entityType"`
	EntityID       string         `bson:"entityId"       json:"entityId"`
	Before         map[string]any `bson:"before"         json:"before,omitempty"`
	After          map[string]any `bson:"after"          json:"after,omitempty"`
	ChangedFields  []string       `bson:"changedFields"  json:"changedFields,omitempty"`
	Risk           Risk           `bson:"risk"           json:"risk"`
	Note           string         `bson:"note"           json:"note,omitempty"`
	At             time.Time      `bson:"at"             json:"at"`
}

// Recorder is the port every module depends on. Implementations must be
// best-effort: an audit failure is logged, never returned, because losing the
// user's actual work to a logging problem is the worse outcome.
type Recorder interface {
	Record(ctx context.Context, e Entry)
}

// highRiskActions are the access- and data-moving changes worth watching. Kept
// as a list here (not at the call sites) so the classification is reviewable in
// one place. These are generic examples — a real project extends them.
var highRiskActions = map[string]struct{}{
	"user.role":          {},
	"user.permissions":   {},
	"user.passwordReset": {},
	"user.delete":        {},
	"apiKey.create":      {},
	"apiKey.revoke":      {},
	"settings.update":    {},
	"tenant.deactivate":  {},
}

// RiskOf classifies an action. Unknown actions are NORMAL — a new action is not
// automatically suspicious, and silently promoting everything to HIGH would make
// the HIGH filter useless.
func RiskOf(action string) Risk {
	if _, ok := highRiskActions[action]; ok {
		return RiskHigh
	}
	return RiskNormal
}

// redactedFields never reach the audit log, whatever a caller passes. Enforced
// here rather than trusted to each call site — one forgetful caller would
// otherwise persist a secret forever in an append-only collection.
var redactedFields = map[string]struct{}{
	"passwordHash": {}, "password": {}, "twoFASecret": {}, "backupCodes": {},
	"accessToken": {}, "refreshToken": {}, "token": {}, "otp": {}, "apiKey": {},
	"secret": {}, "keyHash": {},
}

const redactedPlaceholder = "[redacted]"

// New builds an Entry with the diff computed, secrets stripped and the risk
// classified. before/after may be nil (a create has no before, a delete no after).
func New(action, entityType, entityID string, before, after map[string]any) Entry {
	b, a := Redact(before), Redact(after)
	return Entry{
		ID:            uuid.New(),
		Action:        action,
		EntityType:    entityType,
		EntityID:      entityID,
		Before:        b,
		After:         a,
		ChangedFields: ChangedFields(b, a),
		Risk:          RiskOf(action),
		At:            time.Now().UTC(),
	}
}

// Redact returns a copy with sensitive values replaced. A nil map stays nil so
// "no before state" (a create) is distinguishable from "an empty before state".
func Redact(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		if _, secret := redactedFields[k]; secret {
			out[k] = redactedPlaceholder
			continue
		}
		out[k] = v
	}
	return out
}

// ChangedFields lists the keys whose value differs between before and after,
// sorted for stable output. A key present on only one side counts as changed —
// that is how an added or cleared field shows up in a review.
func ChangedFields(before, after map[string]any) []string {
	if before == nil || after == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(before)+len(after))
	var out []string
	for k := range before {
		seen[k] = struct{}{}
	}
	for k := range after {
		seen[k] = struct{}{}
	}
	for k := range seen {
		bv, inB := before[k]
		av, inA := after[k]
		if inB != inA || !equalValues(bv, av) {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// equalValues compares two field values structurally. Numbers that BSON may have
// round-tripped into a different concrete type (int64 vs float64) must not read
// as a change, or every re-read would look like an edit.
func equalValues(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	an, aok := asFloat(a)
	bn, bok := asFloat(b)
	if aok && bok {
		return an == bn
	}
	// Fall back to formatted comparison for types like time.Time / uuid.UUID
	// whose decoded and in-memory forms differ but print identically.
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}
