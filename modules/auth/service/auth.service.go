package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	roleentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	pkgemail "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	jwtpkg "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/jwt"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

const (
	sessionKeyPrefix = "session:"
	resetKeyPrefix   = "reset:"
	verifyKeyPrefix  = "verify:"
)

type UserRepo interface {
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*entity.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	Create(ctx context.Context, u *entity.User) error
	Update(ctx context.Context, id uuid.UUID, fields bson.M) error
}

type UserSvc interface {
	HandleFailedLogin(ctx context.Context, u *entity.User) error
	RecordLogin(ctx context.Context, id uuid.UUID, ip string) error
}

// RoleRepo is used to resolve the role slug for JWT claims.
type RoleRepo interface {
	FindByID(ctx context.Context, id uuid.UUID) (*roleentity.Role, error)
}

// TenantResolver resolves the tenant slug supplied at login to its id. Optional
// dependency (attached via WithTenants): without it a tenantSlug on the request
// is rejected rather than silently ignored, because ignoring it would let a
// wrong slug appear to work. Returns uuid.Nil + a non-nil error when the slug
// is unknown.
type TenantResolver interface {
	TenantIDBySlug(ctx context.Context, slug string) (uuid.UUID, error)
}

type AuthService struct {
	userRepo    UserRepo
	userSvc     UserSvc
	roleRepo    RoleRepo
	tenants     TenantResolver
	jwtMgr      *jwtpkg.Manager
	hasher      *hash.Hasher
	rdb         *redis.Client
	cfg         config.Config
	localMailer *pkgemail.LocalizedMailer
	log         *zap.Logger
}

// WithTenants attaches the tenant-slug resolver used by tenant-bound login.
// Optional so the constructor signature and every existing caller stay
// unchanged; a request carrying a tenantSlug fails closed when it is absent.
func (s *AuthService) WithTenants(t TenantResolver) *AuthService {
	s.tenants = t
	return s
}

func New(
	userRepo UserRepo,
	userSvc UserSvc,
	roleRepo RoleRepo,
	jwtMgr *jwtpkg.Manager,
	hasher *hash.Hasher,
	rdb *redis.Client,
	cfg config.Config,
	log *zap.Logger,
	localMailer *pkgemail.LocalizedMailer,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		userSvc:     userSvc,
		roleRepo:    roleRepo,
		jwtMgr:      jwtMgr,
		hasher:      hasher,
		rdb:         rdb,
		cfg:         cfg,
		log:         log,
		localMailer: localMailer,
	}
}

// Login authenticates email+password and returns a token pair.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest, ip string) (*dto.LoginResponse, error) {
	u, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.ErrInvalidCredentials
	}
	if !u.IsActive() {
		return nil, errors.ErrAccountLocked
	}
	if u.IsLocked() {
		return nil, errors.ErrAccountLocked
	}
	if !s.hasher.Compare(req.Password, u.PasswordHash) {
		_ = s.userSvc.HandleFailedLogin(ctx, u)
		return nil, errors.ErrInvalidCredentials
	}
	// Checked AFTER the password, deliberately: answering "wrong tenant" to an
	// unauthenticated caller would turn login into an oracle for which tenant an
	// email belongs to.
	if err := s.assertTenantMatches(ctx, req.TenantSlug, u); err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, u, ip)
}

// assertTenantMatches enforces the tenant slug supplied at login. An empty slug
// is accepted unchanged (the JWT's tenant is read from the user record, not the
// request). A platform user (uuid.Nil tenant) has no tenant, so any slug is a
// mismatch for them.
func (s *AuthService) assertTenantMatches(ctx context.Context, slug string, u *entity.User) error {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if slug == "" {
		return nil
	}
	// Fail closed rather than ignore: a silently-dropped slug would let a wrong
	// tenant slug look like it worked.
	if s.tenants == nil {
		return errors.ErrTenantMismatch
	}
	tid, err := s.tenants.TenantIDBySlug(ctx, slug)
	if err != nil || tid == uuid.Nil {
		// Same error as a real mismatch — an unknown slug and a wrong-but-real
		// slug must be indistinguishable, or login re-opens tenant enumeration.
		return errors.ErrTenantMismatch
	}
	if u.TenantID == uuid.Nil || u.TenantID != tid {
		return errors.ErrTenantMismatch
	}
	return nil
}

// Register creates a new user account and returns tokens.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest, ip string) (*dto.LoginResponse, error) {
	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, err
	}
	u := &entity.User{
		Email:         req.Email,
		Username:      req.Username,
		PasswordHash:  passwordHash,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Status:        entity.UserStatusActive,
		EmailVerified: false,
		NotifSettings: entity.DefaultNotifSettings(),
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, u, ip)
}

// Refresh validates a refresh token and issues a new pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*dto.LoginResponse, error) {
	claims, err := s.jwtMgr.ParseRefresh(refreshToken)
	if err != nil {
		return nil, errors.ErrTokenInvalid
	}
	ok, err := s.SessionExists(ctx, claims.SessionID)
	if err != nil || !ok {
		return nil, errors.ErrSessionRevoked
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, errors.ErrUserNotFound
	}
	u, err := s.userRepo.FindByID(ctx, uid)
	if err != nil || u == nil {
		return nil, errors.ErrUserNotFound
	}
	_ = s.RevokeSession(ctx, claims.SessionID)
	return s.issueTokenPair(ctx, u, "")
}

// Logout revokes the caller's session from Redis.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.RevokeSession(ctx, sessionID)
}

// LogoutAll revokes ALL sessions belonging to a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("user_sessions:%s:*", userID)
	iter := s.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		_ = s.RevokeSession(ctx, iter.Val())
	}
	return iter.Err()
}

// Enable2FA generates a TOTP secret for the user.
func (s *AuthService) Enable2FA(ctx context.Context, userID, email string) (*dto.Enable2FAResponse, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.cfg.App.Name,
		AccountName: email,
	})
	if err != nil {
		return nil, err
	}
	backupCodes := make([]string, 8)
	for i := range backupCodes {
		code, _ := hash.RandomHex(5)
		backupCodes[i] = code
	}
	oid, _ := uuid.Parse(userID)
	if err := s.userRepo.Update(ctx, oid, bson.M{
		"twoFASecret": key.Secret(),
		"backupCodes": backupCodes,
	}); err != nil {
		return nil, err
	}
	return &dto.Enable2FAResponse{
		Secret:      key.Secret(),
		QRCodeURL:   key.URL(),
		BackupCodes: backupCodes,
	}, nil
}

// Verify2FA confirms the user-entered TOTP code and activates 2FA.
func (s *AuthService) Verify2FA(ctx context.Context, userID, code, secret string) error {
	if !totp.Validate(code, secret) {
		return errors.ErrInvalidCredentials
	}
	oid, _ := uuid.Parse(userID)
	return s.userRepo.Update(ctx, oid, bson.M{"twoFAEnabled": true})
}

// ── Session helpers ───────────────────────────────────────────────────────────

func (s *AuthService) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	n, err := s.rdb.Exists(ctx, sessionKeyPrefix+sessionID).Result()
	return n > 0, err
}

func (s *AuthService) RevokeSession(ctx context.Context, sessionID string) error {
	return s.rdb.Del(ctx, sessionKeyPrefix+sessionID).Err()
}

// ── Internal ──────────────────────────────────────────────────────────────────

// unionPermissions appends extras that the role does not already grant, keeping
// the role's order first and dropping duplicates.
func unionPermissions(rolePerms, extras []string) []string {
	if len(extras) == 0 {
		return rolePerms
	}
	seen := make(map[string]struct{}, len(rolePerms)+len(extras))
	out := make([]string, 0, len(rolePerms)+len(extras))
	for _, p := range append(append([]string{}, rolePerms...), extras...) {
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

// effectivePermissions resolves what actually reaches the JWT:
//
//	(role ∪ extraPermissions) − revokedPermissions
//
// THE resolver, and the single place that decides what a person can do — a
// second copy elsewhere would be a second answer.
//
// A WILDCARD ROLE IS NOT SUBTRACTABLE, and that branch is the whole safety of
// the feature. "*" is not an enumeration of permissions, it is a claim that
// every gate passes — so "revoke X from an admin" has no coherent meaning:
// subtracting a slug from a list containing only "*" removes nothing, while
// expanding "*" into a concrete set to subtract from would silently demote the
// holder to whatever that set happened to contain. Both readings are wrong, so
// the wildcard is returned untouched; taking capability from a wildcard holder
// is a role change, not a checkbox.
func effectivePermissions(rolePerms, extras, revoked []string) []string {
	perms := unionPermissions(rolePerms, extras)
	if len(revoked) == 0 {
		return perms
	}
	for _, p := range perms {
		if p == string(roleentity.PermAll) {
			return perms
		}
	}
	drop := make(map[string]struct{}, len(revoked))
	for _, p := range revoked {
		drop[p] = struct{}{}
	}
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		if _, gone := drop[p]; gone {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (s *AuthService) issueTokenPair(ctx context.Context, u *entity.User, ip string) (*dto.LoginResponse, error) {
	sessionID := uuid.NewString()
	// Resolve the role slug (e.g. "admin") + its permission set for the JWT
	// claims. RoleProtected compares against slugs; RequirePermission checks the
	// flattened permission list.
	roleSlug := "user" // safe default
	var permissions []string
	if r, err := s.roleRepo.FindByID(ctx, u.RoleID); err == nil {
		roleSlug = string(r.Slug)
		permissions = make([]string, 0, len(r.Permissions))
		for _, p := range r.Permissions {
			permissions = append(permissions, string(p))
		}
	}
	// Effective permissions = (role ∪ per-user grants) − per-user withdrawals.
	permissions = effectivePermissions(permissions, u.ExtraPermissions, u.RevokedPermissions)
	// tenantID is "" for a platform-scoped user (uuid.Nil).
	tenantID := ""
	if u.TenantID != uuid.Nil {
		tenantID = u.TenantID.String()
	}

	accessToken, accessExp, err := s.jwtMgr.IssueAccess(u.ID.String(), sessionID, roleSlug, tenantID, permissions)
	if err != nil {
		return nil, err
	}
	refreshToken, _, err := s.jwtMgr.IssueRefresh(u.ID.String(), sessionID, roleSlug, tenantID, permissions)
	if err != nil {
		return nil, err
	}
	if err := s.rdb.Set(ctx, sessionKeyPrefix+sessionID, u.ID.String(), s.cfg.JWT.RefreshExpire).Err(); err != nil {
		return nil, err
	}
	if ip != "" {
		_ = s.userSvc.RecordLogin(ctx, u.ID, ip)
	}
	expiresIn := int64(time.Until(accessExp).Seconds())
	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		User: dto.UserPayload{
			ID:    u.ID.String(),
			Email: u.Email,
			Role:  roleSlug,
		},
	}, nil
}
