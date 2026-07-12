package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
)

// ── Tests ─────────────────────────────────────────────────────────────────────
// Handlers in this codebase always bind through validate.BindJSON (never
// c.ShouldBindJSON directly — see CLAUDE.md), which reports missing/invalid
// required fields as 422 with error details, not gin's raw 400 bind error.

func TestLogin_MissingBody_Returns422(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/v1/auth/login", func(c *gin.Context) {
		var req dto.LoginRequest
		if !validate.BindJSON(c, &req) {
			return
		}
		c.JSON(http.StatusOK, nil)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_InvalidEmailFormat_Returns422(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/v1/auth/login", func(c *gin.Context) {
		var req dto.LoginRequest
		if !validate.BindJSON(c, &req) {
			return
		}
		c.JSON(http.StatusOK, nil)
	})

	body, _ := json.Marshal(map[string]string{"email": "not-an-email", "password": "pass1234"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHash_RoundTrip(t *testing.T) {
	h := hash.New(4) // low cost for speed
	plain := "MySecurePass@123"

	hashed, err := h.Hash(plain)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if hashed == plain {
		t.Error("hash must differ from plaintext")
	}
	if !h.Compare(plain, hashed) {
		t.Error("Compare should return true for correct password")
	}
	if h.Compare("wrong", hashed) {
		t.Error("Compare should return false for wrong password")
	}
}

func TestErrors_SentinelCodes(t *testing.T) {
	cases := []struct {
		err  *errors.AppError
		code string
		want int
	}{
		{errors.ErrNotFound, "NOT_FOUND", 404},
		{errors.ErrUnauthorized, "UNAUTHORIZED", 401},
		{errors.ErrForbidden, "FORBIDDEN", 403},
		{errors.ErrConflict, "CONFLICT", 409},
		{errors.ErrEmailTaken, "EMAIL_TAKEN", 409},
		{errors.ErrUserNotFound, "USER_NOT_FOUND", 404},
		{errors.ErrAPIKeyInvalid, "API_KEY_INVALID", 401},
		{errors.ErrTokenExpired, "TOKEN_EXPIRED", 401},
	}
	for _, tc := range cases {
		if tc.err.Code != tc.code {
			t.Errorf("%s: code want %q got %q", tc.err.Message, tc.code, tc.err.Code)
		}
		if tc.err.Status != tc.want {
			t.Errorf("%s: status want %d got %d", tc.err.Message, tc.want, tc.err.Status)
		}
	}
}

func TestConfig_Defaults(t *testing.T) {
	// Verify config loads without a .env file (all defaults)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if cfg.App.Port == "" {
		t.Error("APP_PORT should have a default")
	}
	if cfg.Auth.MaxPasswordAttempts == 0 {
		t.Error("AUTH_MAX_PASSWORD_ATTEMPTS should have a default")
	}
	if cfg.Rate.Requests == 0 {
		t.Error("RATE_LIMIT_REQUESTS should have a default")
	}
	if cfg.WebAuthn.RPID == "" {
		t.Error("WEBAUTHN_RP_ID should have a default")
	}
	if cfg.I18n.LocalesDir == "" {
		t.Error("LOCALES_DIR should have a default")
	}
	t.Logf("port=%s env=%s rpid=%s locales=%s", cfg.App.Port, cfg.App.Env, cfg.WebAuthn.RPID, cfg.I18n.LocalesDir)
}
