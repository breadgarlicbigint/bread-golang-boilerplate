package service

import (
	"context"
	"time"

	roleentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

//go:generate mockgen -source=user.service.go -destination=../mocks/user.service.mock.go

type UserRepo interface {
	Create(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	List(ctx context.Context, q pagination.Query) ([]*entity.User, int64, error)
	Update(ctx context.Context, id uuid.UUID, fields bson.M) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	IncrementPasswordAttempts(ctx context.Context, id uuid.UUID) error
	ResetPasswordAttempts(ctx context.Context, id uuid.UUID) error
}

// EmailQueue enqueues a pre-rendered email for the background worker to deliver.
// *worker.Client satisfies this; keeping it an interface avoids importing asynq
// into the service and lets tests pass a fake.
type EmailQueue interface {
	EnqueueEmail(to, subject, html, text string) error
}

// RoleResolver resolves a role id to its slug, for the assignable-role guard.
type RoleResolver interface {
	SlugByID(ctx context.Context, id uuid.UUID) (string, error)
}

// Actor is who is performing a user-management action, for the self-change and
// assignable-role guards. HasWildcard is true for a holder of the "*"
// permission (a platform/super admin), who may assign any role.
type Actor struct {
	ID          string
	HasWildcard bool
}

type UserService struct {
	repo        UserRepo
	roles       RoleResolver
	resetTokens ResetTokenIssuer
	hasher      *hash.Hasher
	cfg         config.AuthConfig
	localMailer *email.LocalizedMailer
	emailQueue  EmailQueue
	appURL      string
	log         *zap.Logger
}

func New(repo UserRepo, hasher *hash.Hasher, cfg config.AuthConfig, localMailer *email.LocalizedMailer, emailQueue EmailQueue, appURL string, log *zap.Logger) *UserService {
	return &UserService{repo: repo, hasher: hasher, cfg: cfg, localMailer: localMailer, emailQueue: emailQueue, appURL: appURL, log: log}
}

// WithRoles attaches the role resolver used by the assignable-role guard.
// Optional so the constructor and existing callers stay unchanged; without it a
// non-wildcard actor cannot assign any role (fails closed).
func (s *UserService) WithRoles(r RoleResolver) *UserService {
	s.roles = r
	return s
}

// validatePermissionGrant enforces the per-user grant rules: every slug must be
// grantable (known, non-wildcard), and no slug may appear in both the grant and
// withdrawal lists (which is ambiguous, not silently ordered).
func validatePermissionGrant(extra, revoked []string) error {
	for _, p := range append(append([]string{}, extra...), revoked...) {
		if !roleentity.IsGrantablePermission(p) {
			return errors.ErrInvalidPermissionGrant
		}
	}
	ex := make(map[string]struct{}, len(extra))
	for _, p := range extra {
		ex[p] = struct{}{}
	}
	for _, p := range revoked {
		if _, both := ex[p]; both {
			return errors.ErrInvalidPermissionGrant
		}
	}
	return nil
}

// guardAssignableRole blocks a non-wildcard actor from assigning a role outside
// the allowlisted set — otherwise an admin could post the wildcard role's id and
// mint a wildcard user. A wildcard actor may assign anything.
func (s *UserService) guardAssignableRole(ctx context.Context, actor Actor, roleID uuid.UUID) error {
	if actor.HasWildcard {
		return nil
	}
	if s.roles == nil {
		return errors.ErrRoleNotAssignable
	}
	slug, err := s.roles.SlugByID(ctx, roleID)
	if err != nil {
		return errors.ErrRoleNotFound
	}
	if !roleentity.IsAssignableTenantRole(roleentity.RoleSlug(slug)) {
		return errors.ErrRoleNotAssignable
	}
	return nil
}

func (s *UserService) Create(ctx context.Context, lang string, req dto.CreateUserRequest, actor Actor) (*entity.User, error) {
	emailTaken, err := s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if emailTaken {
		return nil, errors.ErrEmailTaken
	}

	usernameTaken, err := s.repo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if usernameTaken {
		return nil, errors.ErrUsernameTaken
	}

	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		return nil, errors.ErrBadRequest
	}
	if err := s.guardAssignableRole(ctx, actor, roleID); err != nil {
		return nil, err
	}
	if err := validatePermissionGrant(req.ExtraPermissions, req.RevokedPermissions); err != nil {
		return nil, err
	}

	u := &entity.User{
		Email:              req.Email,
		Username:           req.Username,
		PasswordHash:       passwordHash,
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		PhoneNumber:        req.PhoneNumber,
		RoleID:             roleID,
		ExtraPermissions:   req.ExtraPermissions,
		RevokedPermissions: req.RevokedPermissions,
		Status:             entity.UserStatusActive,
		EmailVerified:      false,
		NotifSettings:      entity.DefaultNotifSettings(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	s.sendWelcomeEmail(ctx, lang, u)

	return u, nil
}

// sendWelcomeEmail renders the localized welcome email and hands it off to the
// background worker queue for delivery. Rendering happens here (fast, in-memory,
// and where the request's lang is known); the retryable network send is done by
// the worker process. Best-effort — a failure here must never roll back or fail
// the user creation itself.
func (s *UserService) sendWelcomeEmail(_ context.Context, lang string, u *entity.User) {
	if s.localMailer == nil || s.emailQueue == nil {
		return // email or worker queue not configured — skip silently
	}
	name := u.FirstName + " " + u.LastName
	rendered, err := s.localMailer.RenderWelcome(lang, name, s.appURL+"/dashboard", s.appURL+"/docs")
	if err != nil {
		if s.log != nil {
			s.log.Warn("user: render welcome email failed", zap.String("userId", u.ID.String()), zap.Error(err))
		}
		return
	}
	subject := s.localMailer.SubjectWelcome(lang, name)
	if err := s.emailQueue.EnqueueEmail(u.Email, subject, rendered.HTML, rendered.Text); err != nil && s.log != nil {
		s.log.Warn("user: enqueue welcome email failed", zap.String("userId", u.ID.String()), zap.Error(err))
	}
}

func (s *UserService) GetByID(ctx context.Context, id string) (*entity.User, error) {
	oid, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.ErrBadRequest
	}
	u, err := s.repo.FindByID(ctx, oid)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.ErrUserNotFound
	}
	return u, nil
}

func (s *UserService) List(ctx context.Context, q pagination.Query) ([]*entity.User, int64, error) {
	return s.repo.List(ctx, q)
}

func (s *UserService) Update(ctx context.Context, id string, req dto.UpdateUserRequest, actor Actor) (*entity.User, error) {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Access changes (role / permissions) are guarded. Nobody changes their OWN
	// role or permissions — on any route, including /me — so a user cannot
	// self-elevate. A non-wildcard admin may assign only an allowlisted role.
	changesAccess := req.RoleID != "" || req.ExtraPermissions != nil || req.RevokedPermissions != nil
	if changesAccess && actor.ID == u.ID.String() {
		return nil, errors.ErrCannotChangeOwnAccess
	}

	fields := bson.M{}
	if req.FirstName != "" {
		fields["firstName"] = req.FirstName
	}
	if req.LastName != "" {
		fields["lastName"] = req.LastName
	}
	if req.PhoneNumber != "" {
		fields["phoneNumber"] = req.PhoneNumber
	}
	if req.ProfilePicture != "" {
		fields["profilePicture"] = req.ProfilePicture
	}
	if req.Gender != "" {
		fields["gender"] = req.Gender
	}
	if req.RoleID != "" {
		roleID, err := uuid.Parse(req.RoleID)
		if err != nil {
			return nil, errors.ErrBadRequest
		}
		if err := s.guardAssignableRole(ctx, actor, roleID); err != nil {
			return nil, err
		}
		fields["roleId"] = roleID
	}
	if req.ExtraPermissions != nil || req.RevokedPermissions != nil {
		extra := u.ExtraPermissions
		if req.ExtraPermissions != nil {
			extra = *req.ExtraPermissions
		}
		revoked := u.RevokedPermissions
		if req.RevokedPermissions != nil {
			revoked = *req.RevokedPermissions
		}
		if err := validatePermissionGrant(extra, revoked); err != nil {
			return nil, err
		}
		fields["extraPermissions"] = extra
		fields["revokedPermissions"] = revoked
	}

	if err := s.repo.Update(ctx, u.ID, fields); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *UserService) ChangePassword(ctx context.Context, id string, req dto.ChangePasswordRequest) error {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !s.hasher.Compare(req.OldPassword, u.PasswordHash) {
		return errors.ErrInvalidCredentials
	}
	if s.hasher.Compare(req.NewPassword, u.PasswordHash) {
		return errors.ErrPasswordSameAsOld
	}
	newHash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, u.ID, bson.M{"passwordHash": newHash})
}

func (s *UserService) BlockUser(ctx context.Context, id, reason string) error {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, u.ID, bson.M{"status": entity.UserStatusBlocked})
}

func (s *UserService) UnblockUser(ctx context.Context, id string) error {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, u.ID, bson.M{"status": entity.UserStatusActive})
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.SoftDelete(ctx, u.ID)
}

// RecordLogin updates last login metadata and resets failure counter.
func (s *UserService) RecordLogin(ctx context.Context, id uuid.UUID, ip string) error {
	now := time.Now()
	return s.repo.Update(ctx, id, bson.M{
		"lastLoginAt":      now,
		"lastLoginIP":      ip,
		"passwordAttempts": 0,
		"lockedUntil":      nil,
	})
}

// HandleFailedLogin increments the attempt counter and locks if threshold hit.
func (s *UserService) HandleFailedLogin(ctx context.Context, u *entity.User) error {
	if err := s.repo.IncrementPasswordAttempts(ctx, u.ID); err != nil {
		return err
	}
	if u.PasswordAttempts+1 >= s.cfg.MaxPasswordAttempts {
		lockUntil := time.Now().Add(s.cfg.LockoutDuration)
		return s.repo.Update(ctx, u.ID, bson.M{"lockedUntil": lockUntil})
	}
	return nil
}

// MapToResponse converts a User entity to a safe DTO (no secrets).
func MapToResponse(u *entity.User) dto.UserResponse {
	return dto.UserResponse{
		ID:             u.ID,
		Email:          u.Email,
		Username:       u.Username,
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		PhoneNumber:    u.PhoneNumber,
		ProfilePicture: u.ProfilePicture,
		Gender:         string(u.Gender),
		Status:         string(u.Status),
		RoleID:         u.RoleID,
		EmailVerified:  u.EmailVerified,
		TwoFAEnabled:   u.TwoFAEnabled,
		LastLoginAt:    u.LastLoginAt,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
	}
}
