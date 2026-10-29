package audit_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/audit"
)

// The diff is what a reviewer reads when settling a dispute, so it has to be
// exact: no phantom changes from BSON number widening, and no missed ones.
func TestChangedFields(t *testing.T) {
	cases := []struct {
		name          string
		before, after map[string]any
		want          []string
	}{
		{"changed value",
			map[string]any{"price": int64(45000), "name": "X"},
			map[string]any{"price": int64(39000), "name": "X"},
			[]string{"price"}},
		{"same number, different numeric type",
			map[string]any{"qty": int64(5)},
			map[string]any{"qty": float64(5)},
			nil},
		{"field added",
			map[string]any{"name": "x"},
			map[string]any{"name": "x", "code": "123"},
			[]string{"code"}},
		{"field cleared",
			map[string]any{"name": "x", "code": "123"},
			map[string]any{"name": "x"},
			[]string{"code"}},
		{"multiple changes are sorted",
			map[string]any{"b": 1, "a": 1, "c": 1},
			map[string]any{"b": 2, "a": 2, "c": 1},
			[]string{"a", "b"}},
		{"nil before (create)", nil, map[string]any{"name": "x"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := audit.ChangedFields(tc.before, tc.after); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ChangedFields = %v, want %v", got, tc.want)
			}
		})
	}
}

// Redaction is enforced centrally: one forgetful call site would otherwise
// persist a secret forever in an append-only collection.
func TestRedact(t *testing.T) {
	in := map[string]any{
		"email": "a@b.test", "passwordHash": "$2a$12$real", "twoFASecret": "ABC",
		"backupCodes": []string{"1", "2"}, "refreshToken": "tok", "name": "Alice",
	}
	out := audit.Redact(in)

	for _, k := range []string{"passwordHash", "twoFASecret", "backupCodes", "refreshToken"} {
		if out[k] != "[redacted]" {
			t.Errorf("%s = %v, want it redacted", k, out[k])
		}
	}
	if out["email"] != "a@b.test" || out["name"] != "Alice" {
		t.Error("non-secret fields must survive — the trail is useless without them")
	}
	if in["passwordHash"] != "$2a$12$real" {
		t.Error("Redact must not mutate its input")
	}
	if audit.Redact(nil) != nil {
		t.Error("nil in ⇒ nil out")
	}
}

func TestRiskOf(t *testing.T) {
	for _, a := range []string{"user.role", "user.permissions", "apiKey.create", "apiKey.revoke", "settings.update"} {
		if audit.RiskOf(a) != audit.RiskHigh {
			t.Errorf("%s should be HIGH risk — it is an access/data lever", a)
		}
	}
	// An unknown action is NORMAL: promoting everything to HIGH would make the
	// HIGH filter worthless.
	for _, a := range []string{"user.create", "whatever.new"} {
		if audit.RiskOf(a) != audit.RiskNormal {
			t.Errorf("%s should be NORMAL risk", a)
		}
	}
}

func TestNew_ComputesDiffAndRedacts(t *testing.T) {
	e := audit.New("user.role", "users", "u1",
		map[string]any{"role": "user", "passwordHash": "x"},
		map[string]any{"role": "admin", "passwordHash": "x"})

	if e.Risk != audit.RiskHigh {
		t.Errorf("risk = %s, want HIGH for user.role", e.Risk)
	}
	if !reflect.DeepEqual(e.ChangedFields, []string{"role"}) {
		t.Errorf("changedFields = %v, want just role", e.ChangedFields)
	}
	if e.Before["passwordHash"] != "[redacted]" || e.After["passwordHash"] != "[redacted]" {
		t.Error("secrets must be redacted on both sides")
	}
	if e.At.IsZero() {
		t.Error("entry should be stamped with a time")
	}
}

func TestAfterDelete(t *testing.T) {
	at := time.Now()
	before := map[string]any{"id": "u1", "name": "Alice", "role": "user"}
	after := audit.AfterDelete(before, "name", at)

	if after["name"] != "Alice [Deleted]" {
		t.Errorf("name = %v, want %q", after["name"], "Alice [Deleted]")
	}
	if after["deletedAt"] != at || after["updatedAt"] != at {
		t.Error("deletedAt/updatedAt must be stamped")
	}
	if before["name"] != "Alice" {
		t.Error("AfterDelete must not mutate its input")
	}
	if audit.AfterDelete(nil, "name", at) != nil {
		t.Error("nil before ⇒ nil after")
	}
}
