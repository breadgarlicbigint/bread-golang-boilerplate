package validate_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	pkgi18n "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
)

func init() { gin.SetMode(gin.TestMode) }

// ── test DTOs ─────────────────────────────────────────────────────────────────

type loginReq struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type createReq struct {
	Name  string `json:"name"  validate:"required,min=2,max=50"`
	Age   int    `json:"age"   validate:"required,gte=18,lte=120"`
	Phone string `json:"phone" validate:"omitempty,e164"`
	Role  string `json:"role"  validate:"required,oneof=admin user member"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func newCtx(body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	b, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBuffer(b))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func responseBody(w *httptest.ResponseRecorder) map[string]interface{} {
	var m map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

// newCtxLang binds the real locales/ translator to the context, exactly as the
// i18n middleware does in apps/api.
func newCtxLang(t *testing.T, lang string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	tr, err := pkgi18n.New(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatalf("load locales: %v", err)
	}
	c, w := newCtx(body)
	c.Set(pkgi18n.CtxLang, lang)
	c.Set(pkgi18n.CtxTranslator, tr)
	return c, w
}

func messages(w *httptest.ResponseRecorder) map[string]string {
	out := map[string]string{}
	body := responseBody(w)
	errs, _ := body["errors"].([]interface{})
	for _, e := range errs {
		d := e.(map[string]interface{})
		field, _ := d["field"].(string)
		msg, _ := d["message"].(string)
		out[field] = msg
	}
	return out
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestBindJSON_ValidPayload_ReturnsTrue(t *testing.T) {
	c, _ := newCtx(map[string]interface{}{
		"email": "alice@example.com", "password": "StrongPass1",
	})
	var req loginReq
	if !validate.BindJSON(c, &req) {
		t.Error("expected true for valid payload")
	}
	if req.Email != "alice@example.com" {
		t.Errorf("expected email to be bound, got %q", req.Email)
	}
}

func TestBindJSON_MissingRequiredField_Returns422WithDetails(t *testing.T) {
	c, w := newCtx(map[string]interface{}{"email": "alice@example.com"}) // missing password
	var req loginReq
	if validate.BindJSON(c, &req) {
		t.Error("expected false for missing required field")
	}
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
	body := responseBody(w)
	errs, ok := body["errors"].([]interface{})
	if !ok || len(errs) == 0 {
		t.Fatal("expected errors[] with field details")
	}
	first := errs[0].(map[string]interface{})
	if first["field"] == nil {
		t.Error("expected 'field' key in error detail")
	}
	if first["message"] == nil {
		t.Error("expected 'message' key in error detail")
	}
}

func TestBindJSON_InvalidEmail_Returns422WithFieldName(t *testing.T) {
	c, w := newCtx(map[string]interface{}{"email": "not-an-email", "password": "StrongPass1"})
	var req loginReq
	validate.BindJSON(c, &req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
	if got := messages(w)["email"]; got == "" {
		t.Error("expected an error keyed on field=email")
	}
}

func TestBindJSON_MalformedJSON_Returns400(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{invalid json`))
	c.Request.Header.Set("Content-Type", "application/json")

	var req loginReq
	if validate.BindJSON(c, &req) {
		t.Error("expected false for malformed JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestBindJSON_MultipleErrors_ReturnsAllFields(t *testing.T) {
	c, w := newCtx(map[string]interface{}{
		"name": "x",          // too short (min=2)
		"age":  15,           // below gte=18
		"role": "superadmin", // not in oneof
	})
	var req createReq
	validate.BindJSON(c, &req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
	if errs, _ := responseBody(w)["errors"].([]interface{}); len(errs) < 2 {
		t.Errorf("expected multiple field errors, got %d", len(errs))
	}
}

func TestBindJSON_Oneof_HumanMessage(t *testing.T) {
	c, w := newCtx(map[string]interface{}{"name": "Alice", "age": 25, "role": "superadmin"})
	var req createReq
	validate.BindJSON(c, &req)

	msg := messages(w)["role"]
	if msg == "oneof" || msg == "" {
		t.Errorf("expected human message, got %q", msg)
	}
	// Must list the allowed values, comma-separated by friendlyParam.
	if !strings.Contains(msg, "admin, user, member") {
		t.Errorf("oneof message should list allowed values: %q", msg)
	}
}

// ── kind-aware comparison messages ─────────────────────────────────────────────

// numericReq exercises the kind-aware comparison messages: `min`/`max` on an int
// is a bound, not a character count.
type numericReq struct {
	Year int    `json:"year" validate:"required,min=1900,max=2100"`
	ID   string `json:"id"   validate:"omitempty,uuid"`
}

// Regression: `min=1900` on an int used to render "must be at least 1900
// characters long" — the wrong noun.
func TestBindJSON_NumericBound_UsesNumberMessageNotCharacters(t *testing.T) {
	c, w := newCtxLang(t, "en", map[string]interface{}{"year": 1800})
	var req numericReq
	validate.BindJSON(c, &req)

	msg := messages(w)["year"]
	if strings.Contains(msg, "characters") {
		t.Errorf("numeric bound rendered as a length: %q", msg)
	}
	if !strings.Contains(msg, "1900") {
		t.Errorf("expected the bound in the message, got %q", msg)
	}
}

// uuid had no locale key and fell through to a hardcoded English string.
func TestBindJSON_UUID_ResolvesFromLocale(t *testing.T) {
	c, w := newCtxLang(t, "en", map[string]interface{}{"year": 2020, "id": "not-a-uuid"})
	var req numericReq
	validate.BindJSON(c, &req)

	if msg := messages(w)["id"]; !strings.Contains(msg, "UUID") {
		t.Errorf("uuid message not resolved: %q", msg)
	}
}

// Any tag without a dedicated key must still resolve through validation.default —
// and the default must not name the validator tag, which is an internal detail.
func TestBindJSON_UnknownTag_FallsBackToDefault(t *testing.T) {
	type oddReq struct {
		Slug string `json:"name" validate:"required,startswith=SKU-"`
	}
	c, w := newCtxLang(t, "en", map[string]interface{}{"name": "ABC"})
	var req oddReq
	validate.BindJSON(c, &req)

	msg := messages(w)["name"]
	if msg == "" || strings.Contains(msg, "startswith") {
		t.Errorf("default message leaks the validator tag or is empty: %q", msg)
	}
	if !strings.Contains(msg, "is not valid") {
		t.Errorf("expected validation.default, got %q", msg)
	}
}

// ── no internal identifiers in UI text ────────────────────────────────────────

type labelReq struct {
	PhoneNumber string `json:"phoneNumber" validate:"required"`
	Username    string `json:"username"    validate:"omitempty,min=3"`
}

func TestBindJSON_MessageUsesHumanLabelNotJSONTag(t *testing.T) {
	c, w := newCtxLang(t, "en", map[string]interface{}{"username": "ab"})
	var req labelReq
	validate.BindJSON(c, &req)

	msgs := messages(w)
	want := map[string]string{
		"phoneNumber": "Phone number is required",
		"username":    "Username must be at least 3 characters long",
	}
	for field, expected := range want {
		if msgs[field] != expected {
			t.Errorf("%s: want %q, got %q", field, expected, msgs[field])
		}
	}
	// The wire identifier still keys the detail — the clients render the message
	// under it, and it must not change.
	if _, ok := msgs["phoneNumber"]; !ok {
		t.Error("expected the raw JSON tag to survive as the detail's field key")
	}
}

// The identifiers must not appear anywhere in the prose.
func TestBindJSON_MessageNeverContainsAnIdentifier(t *testing.T) {
	c, w := newCtxLang(t, "en", map[string]interface{}{"username": "ab"})
	var req labelReq
	validate.BindJSON(c, &req)

	for field, msg := range messages(w) {
		for _, leak := range []string{field, "labelReq", "json"} {
			if strings.Contains(msg, leak) {
				t.Errorf("%s message %q exposes %q", field, msg, leak)
			}
		}
	}
}

// A type mismatch used to answer with Go's decoder text — struct name, field and
// Go type — straight to the UI.
func TestBindJSON_TypeMismatch_DoesNotLeakGoStructOrType(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/",
		bytes.NewBufferString(`{"year":"not-a-number"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	var req numericReq
	if validate.BindJSON(c, &req) {
		t.Fatal("expected false for a type mismatch")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}

	msg, _ := responseBody(w)["message"].(string)
	for _, leak := range []string{"numericReq", "Go struct", "of type", "int", "unmarshal", "year"} {
		if strings.Contains(msg, leak) {
			t.Errorf("400 body %q exposes %q", msg, leak)
		}
	}
	if msg == "" {
		t.Error("expected the generic invalid-body message, got an empty message")
	}
}

// No translator bound (worker paths, unit tests) must still yield English prose.
func TestBindJSON_NoTranslator_FallsBackToEnglish(t *testing.T) {
	c, w := newCtx(map[string]interface{}{"email": "alice@example.com"})
	var req loginReq
	validate.BindJSON(c, &req)

	if msg := messages(w)["password"]; msg != "Password is required" {
		t.Errorf("want English fallback, got %q", msg)
	}
}

// The English fallback map must not drift from locales/en.json.
func TestEnglishFallbacks_MatchEnLocale(t *testing.T) {
	tr, err := pkgi18n.New(filepath.Join("..", "..", "locales"))
	if err != nil {
		t.Fatalf("load locales: %v", err)
	}
	tags := []string{
		"required", "email", "uuid", "url", "e164", "alphanum", "alpha", "numeric",
		"oneof", "contains", "unique", "notblank", "default",
		"minLength", "maxLength", "lenLength", "gtLength", "gteLength", "ltLength", "lteLength",
		"minNumber", "maxNumber", "lenNumber", "gtNumber", "gteNumber", "ltNumber", "lteNumber",
		"minItems", "maxItems", "lenItems", "gtItems", "gteItems", "ltItems", "lteItems",
	}
	data := map[string]string{"Field": "Name", "Param": "5"}
	for _, tag := range tags {
		key := "validation." + tag
		got := tr.T("en", key, data)
		if got == key {
			t.Errorf("locales/en.json is missing %s", key)
			continue
		}
		if want := validate.FallbackMessageForTest(tag, data); got != want {
			t.Errorf("%s drift:\n  en.json  = %q\n  fallback = %q", key, got, want)
		}
	}
}

func TestBindJSON_EmptyBody_Returns400(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/", bytes.NewBufferString(""))
	c.Request.Header.Set("Content-Type", "application/json")

	var req loginReq
	if validate.BindJSON(c, &req) {
		t.Error("expected false for empty body")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ── dived scalar slices ───────────────────────────────────────────────────────

// go-playground reports a `dive`d element by an INDEXED name — `userIds[0]`, not
// `userIds` — and both the label lookup and the client's per-field mapping break
// on it, so baseField strips the index.
type divedReq struct {
	UserIDs []string `json:"userIds" validate:"omitempty,dive,uuid"`
}

func TestBindJSON_DivedElement_ReportsTheBaseFieldName(t *testing.T) {
	c, w := newCtxLang(t, "en", map[string]interface{}{"userIds": []string{"not-a-uuid"}})
	var req divedReq
	if validate.BindJSON(c, &req) {
		t.Fatal("expected false for an invalid dived element")
	}
	msgs := messages(w)
	if _, ok := msgs["userIds"]; !ok {
		t.Errorf("expected the base field name userIds, got keys %v", msgs)
	}
	if _, ok := msgs["userIds[0]"]; ok {
		t.Error("errors[].field still carries the array index")
	}
	// And the message names it by its label, not an indexed identifier.
	if msg := msgs["userIds"]; strings.Contains(msg, "[0]") {
		t.Errorf("message carries an array index: %q", msg)
	}
}

// notblank rejects whitespace-only strings that required/min accept.
func TestBindJSON_NotBlank_RejectsWhitespace(t *testing.T) {
	type reasonReq struct {
		Reason string `json:"reason" validate:"required,notblank"`
	}
	c, w := newCtxLang(t, "en", map[string]interface{}{"reason": "   "})
	var req reasonReq
	if validate.BindJSON(c, &req) {
		t.Fatal("expected false for a whitespace-only notblank field")
	}
	if msg := messages(w)["reason"]; msg == "" {
		t.Error("expected a notblank error for reason")
	}
}
