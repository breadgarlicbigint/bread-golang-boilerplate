package response

import (
	"encoding/json"
	"testing"
)

// The error envelope must stay byte-identical for an error that sets no code,
// no field detail and no structured details — Code, Details and Errors are all
// omitempty, so nothing new appears unless a caller opts in. A client parsing
// the old shape must not break on the new fields.
func TestErrorEnvelope_OmitsNewFieldsWhenUnset(t *testing.T) {
	env := ErrorEnvelope{
		StatusCode: 404,
		Message:    "Resource not found",
		Timestamp:  "2026-01-01T00:00:00Z",
		Path:       "/v1/users/x",
		RequestID:  "req-1",
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"statusCode":404,"message":"Resource not found","timestamp":"2026-01-01T00:00:00Z","path":"/v1/users/x","requestId":"req-1"}`
	if string(b) != want {
		t.Errorf("envelope drifted:\n got  %s\n want %s", b, want)
	}
}

// When a caller DOES set code and details they appear; this pins the field names
// clients branch on.
func TestErrorEnvelope_CarriesCodeAndDetailsWhenSet(t *testing.T) {
	env := ErrorEnvelope{
		StatusCode: 409,
		Code:       "EMAIL_TAKEN",
		Message:    "Email is already registered",
		Details:    map[string]any{"conflictingId": "abc"},
		Timestamp:  "2026-01-01T00:00:00Z",
		Path:       "/v1/users",
		RequestID:  "req-2",
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round map[string]any
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatal(err)
	}
	if round["code"] != "EMAIL_TAKEN" {
		t.Errorf("code = %v, want EMAIL_TAKEN", round["code"])
	}
	if _, ok := round["details"]; !ok {
		t.Error("details missing from envelope")
	}
}
