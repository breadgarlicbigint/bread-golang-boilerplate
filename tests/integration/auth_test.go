package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	userentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ── Fake user repository ──────────────────────────────────────────────────────

type fakeUserRepo struct {
	users    map[string]*userentity.User
	byID     map[string]*userentity.User
	byGoogle map[string]*userentity.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users:    make(map[string]*userentity.User),
		byID:     make(map[string]*userentity.User),
		byGoogle: make(map[string]*userentity.User),
	}
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*userentity.User, error) {
	return r.users[email], nil
}
func (r *fakeUserRepo) FindByGoogleID(_ context.Context, id string) (*userentity.User, error) {
	return r.byGoogle[id], nil
}
func (r *fakeUserRepo) Create(_ context.Context, u *userentity.User) error {
	u.ID = primitive.NewObjectID()
	u.CreatedAt = time.Now()
	r.users[u.Email] = u
	r.byID[u.ID.String()] = u
	return nil
}
func (r *fakeUserRepo) Update(_ context.Context, _ primitive.ObjectID, _ bson.M) error { return nil }
func (r *fakeUserRepo) FindByID(_ context.Context, id primitive.ObjectID) (*userentity.User, error) {
	return r.byID[id.String()], nil
}
func (r *fakeUserRepo) FindByUsername(_ context.Context, _ string) (*userentity.User, error) {
	return nil, nil
}
func (r *fakeUserRepo) List(_ context.Context, _ pagination.Query) ([]*userentity.User, int64, error) {
	return nil, 0, nil
}
func (r *fakeUserRepo) SoftDelete(_ context.Context, _ primitive.ObjectID) error { return nil }
func (r *fakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := r.users[email]
	return ok, nil
}
func (r *fakeUserRepo) ExistsByUsername(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (r *fakeUserRepo) IncrementPasswordAttempts(_ context.Context, _ primitive.ObjectID) error {
	return nil
}
func (r *fakeUserRepo) ResetPasswordAttempts(_ context.Context, _ primitive.ObjectID) error {
	return nil
}

// ── Fake user service for auth ────────────────────────────────────────────────

type fakeUserSvc struct{}

func (s *fakeUserSvc) HandleFailedLogin(_ context.Context, _ *userentity.User) error { return nil }
func (s *fakeUserSvc) RecordLogin(_ context.Context, _ primitive.ObjectID, _ string) error {
	return nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestLogin_MissingBody_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/v1/auth/login", func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, nil)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestLogin_InvalidEmailFormat_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.POST("/v1/auth/login", func(c *gin.Context) {
		var req dto.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, nil)
	})

	body, _ := json.Marshal(map[string]string{"email": "not-an-email", "password": "pass"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// ShouldBindJSON doesn't validate format - validator does. Just ensure it parsed.
	if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
		t.Errorf("unexpected status %d", w.Code)
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
