package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	userSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"go.mongodb.org/mongo-driver/bson"
)

// ── Fake repo ─────────────────────────────────────────────────────────────────

type fakeUserRepo struct {
	byID       map[string]*entity.User
	byEmail    map[string]*entity.User
	byUsername map[string]*entity.User
}

func newFakeRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:       make(map[string]*entity.User),
		byEmail:    make(map[string]*entity.User),
		byUsername: make(map[string]*entity.User),
	}
}

func (r *fakeUserRepo) Create(_ context.Context, u *entity.User) error {
	u.ID = uuid.New()
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	r.byID[u.ID.String()] = u
	r.byEmail[u.Email] = u
	r.byUsername[u.Username] = u
	return nil
}

func (r *fakeUserRepo) FindByID(_ context.Context, id uuid.UUID) (*entity.User, error) {
	u, ok := r.byID[id.String()]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*entity.User, error) {
	return r.byEmail[email], nil
}

func (r *fakeUserRepo) FindByUsername(_ context.Context, username string) (*entity.User, error) {
	return r.byUsername[username], nil
}

func (r *fakeUserRepo) List(_ context.Context, _ pagination.Query) ([]*entity.User, int64, error) {
	users := make([]*entity.User, 0, len(r.byID))
	for _, u := range r.byID {
		users = append(users, u)
	}
	return users, int64(len(users)), nil
}

func (r *fakeUserRepo) Update(_ context.Context, id uuid.UUID, fields bson.M) error {
	u, ok := r.byID[id.String()]
	if !ok {
		return nil
	}
	if v, ok := fields["status"]; ok {
		if s, ok := v.(entity.UserStatus); ok {
			u.Status = s
		} else {
			u.Status = entity.UserStatus(v.(string))
		}
	}
	if v, ok := fields["passwordHash"]; ok {
		u.PasswordHash = v.(string)
	}
	return nil
}

func (r *fakeUserRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	delete(r.byID, id.String())
	return nil
}

func (r *fakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := r.byEmail[email]
	return ok, nil
}

func (r *fakeUserRepo) ExistsByUsername(_ context.Context, username string) (bool, error) {
	_, ok := r.byUsername[username]
	return ok, nil
}

func (r *fakeUserRepo) IncrementPasswordAttempts(_ context.Context, id uuid.UUID) error {
	if u, ok := r.byID[id.String()]; ok {
		u.PasswordAttempts++
	}
	return nil
}

func (r *fakeUserRepo) ResetPasswordAttempts(_ context.Context, id uuid.UUID) error {
	if u, ok := r.byID[id.String()]; ok {
		u.PasswordAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func newSvc(t *testing.T) (*userSvc.UserService, *fakeUserRepo) {
	t.Helper()
	repo := newFakeRepo()
	hasher := hash.New(4)
	cfg := config.AuthConfig{MaxPasswordAttempts: 5, LockoutDuration: 15 * time.Minute}
	// localMailer, emailQueue, and log are all nil-safe — welcome email sending
	// is a no-op when either is nil (see UserService.sendWelcomeEmail).
	return userSvc.New(repo, hasher, cfg, nil, nil, "http://localhost:5173", nil), repo
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
	svc, _ := newSvc(t)
	req := dto.CreateUserRequest{
		Email:     "alice@example.com",
		Username:  "alice",
		Password:  "Password@123",
		FirstName: "Alice",
		LastName:  "Smith",
		RoleID:    uuid.New().String(),
	}
	u, err := svc.Create(context.Background(), "en", req)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if u.Email != req.Email {
		t.Errorf("email mismatch: want %s got %s", req.Email, u.Email)
	}
	if u.PasswordHash == req.Password {
		t.Error("password must be hashed")
	}
}

func TestCreate_DuplicateEmail(t *testing.T) {
	svc, repo := newSvc(t)
	repo.byEmail["alice@example.com"] = &entity.User{Email: "alice@example.com"}

	_, err := svc.Create(context.Background(), "en", dto.CreateUserRequest{
		Email: "alice@example.com", Username: "alice2", Password: "Password@123",
		FirstName: "A", LastName: "B", RoleID: uuid.New().String(),
	})
	if err == nil {
		t.Fatal("expected ErrEmailTaken")
	}
	ae, ok := errors.As(err)
	if !ok || ae.Code != "EMAIL_TAKEN" {
		t.Errorf("expected EMAIL_TAKEN error, got: %v", err)
	}
}

func TestCreate_DuplicateUsername(t *testing.T) {
	svc, repo := newSvc(t)
	repo.byUsername["alice"] = &entity.User{Username: "alice"}

	_, err := svc.Create(context.Background(), "en", dto.CreateUserRequest{
		Email: "other@example.com", Username: "alice", Password: "Password@123",
		FirstName: "A", LastName: "B", RoleID: uuid.New().String(),
	})
	ae, ok := errors.As(err)
	if !ok || ae.Code != "USERNAME_TAKEN" {
		t.Errorf("expected USERNAME_TAKEN, got: %v", err)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.GetByID(context.Background(), uuid.New().String())
	ae, ok := errors.As(err)
	if !ok || ae.Code != "USER_NOT_FOUND" {
		t.Errorf("expected USER_NOT_FOUND, got: %v", err)
	}
}

func TestGetByID_InvalidObjectID(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.GetByID(context.Background(), "not-an-object-id")
	ae, ok := errors.As(err)
	if !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got: %v", err)
	}
}

func TestBlockAndUnblockUser(t *testing.T) {
	svc, repo := newSvc(t)
	id := uuid.New()
	repo.byID[id.String()] = &entity.User{ID: id, Status: entity.UserStatusActive}

	if err := svc.BlockUser(context.Background(), id.String(), "spam"); err != nil {
		t.Fatal("BlockUser:", err)
	}
	if repo.byID[id.String()].Status != entity.UserStatusBlocked {
		t.Error("expected status blocked")
	}
	if err := svc.UnblockUser(context.Background(), id.String()); err != nil {
		t.Fatal("UnblockUser:", err)
	}
	if repo.byID[id.String()].Status != entity.UserStatusActive {
		t.Error("expected status active")
	}
}

func TestList(t *testing.T) {
	svc, repo := newSvc(t)
	for i := 0; i < 5; i++ {
		id := uuid.New()
		repo.byID[id.String()] = &entity.User{ID: id, Status: entity.UserStatusActive}
	}
	users, total, err := svc.List(context.Background(), pagination.Query{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal("List:", err)
	}
	if total != 5 {
		t.Errorf("total: want 5 got %d", total)
	}
	if len(users) != 5 {
		t.Errorf("len: want 5 got %d", len(users))
	}
}

func TestChangePassword_WrongOld(t *testing.T) {
	svc, repo := newSvc(t)
	hasher := hash.New(4)
	pw, _ := hasher.Hash("correct")
	id := uuid.New()
	repo.byID[id.String()] = &entity.User{ID: id, PasswordHash: pw, Status: entity.UserStatusActive}

	err := svc.ChangePassword(context.Background(), id.String(), dto.ChangePasswordRequest{
		OldPassword: "wrong",
		NewPassword: "NewPassword@123",
	})
	ae, ok := errors.As(err)
	if !ok || ae.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS, got: %v", err)
	}
}
