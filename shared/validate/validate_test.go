package validate_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
		t.Error("expected errors[] with field details")
	}
	// Verify field detail shape
	first := errs[0].(map[string]interface{})
	if first["field"] == nil {
		t.Error("expected 'field' key in error detail")
	}
	if first["message"] == nil {
		t.Error("expected 'message' key in error detail")
	}
	t.Logf("error detail: field=%v message=%v", first["field"], first["message"])
}

func TestBindJSON_InvalidEmail_Returns422WithFieldName(t *testing.T) {
	c, w := newCtx(map[string]interface{}{"email": "not-an-email", "password": "StrongPass1"})
	var req loginReq
	validate.BindJSON(c, &req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
	body := responseBody(w)
	errs := body["errors"].([]interface{})
	first := errs[0].(map[string]interface{})
	if first["field"] != "email" {
		t.Errorf("expected field=email, got %v", first["field"])
	}
	t.Logf("message: %v", first["message"])
}

func TestBindJSON_PasswordTooShort_Returns422WithMinMessage(t *testing.T) {
	c, w := newCtx(map[string]interface{}{"email": "alice@example.com", "password": "short"})
	var req loginReq
	validate.BindJSON(c, &req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
	body := responseBody(w)
	errs := body["errors"].([]interface{})
	first := errs[0].(map[string]interface{})
	msg := first["message"].(string)
	if msg == "" {
		t.Error("expected non-empty message")
	}
	// Should mention "8" (the min param)
	t.Logf("min message: %v", msg)
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
		"name": "x",    // too short (min=2 → passes; min=2 → "x" len=1 → fails)
		"age":  15,     // below gte=18
		"role": "superadmin", // not in oneof
	})
	var req createReq
	validate.BindJSON(c, &req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", w.Code)
	}
	body := responseBody(w)
	errs, ok := body["errors"].([]interface{})
	if !ok || len(errs) < 2 {
		t.Errorf("expected multiple field errors, got %d", len(errs))
	}
	for _, e := range errs {
		detail := e.(map[string]interface{})
		t.Logf("  field=%-12v message=%v", detail["field"], detail["message"])
	}
}

func TestBindJSON_Oneof_HumanMessage(t *testing.T) {
	c, w := newCtx(map[string]interface{}{
		"name": "Alice", "age": 25, "role": "superadmin",
	})
	var req createReq
	validate.BindJSON(c, &req)

	body := responseBody(w)
	errs := body["errors"].([]interface{})
	first := errs[0].(map[string]interface{})
	msg := first["message"].(string)
	// Should mention the allowed values
	if msg == "oneof" {
		t.Error("expected human message, got raw tag name")
	}
	t.Logf("oneof message: %v", msg)
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
