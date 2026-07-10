package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	roleentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	jwtpkg "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/jwt"
	pkgemail "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
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

type AuthService struct {
	userRepo    UserRepo
	userSvc     UserSvc
	roleRepo    RoleRepo
	jwtMgr      *jwtpkg.Manager
	hasher      *hash.Hasher
	rdb         *redis.Client
	cfg         config.Config
	localMailer *pkgemail.LocalizedMailer
	log         *zap.Logger
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
	return s.issueTokenPair(ctx, u, ip)
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
	u, err := s.userRepo.FindByEmail(ctx, claims.UserID)
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

func (s *AuthService) issueTokenPair(ctx context.Context, u *entity.User, ip string) (*dto.LoginResponse, error) {
	sessionID := uuid.NewString()
	// Resolve the role slug (e.g. "admin") for the JWT claim.
	// RoleProtected middleware compares against slugs, not UUIDs.
	roleSlug := "user" // safe default
	if r, err := s.roleRepo.FindByID(ctx, u.RoleID); err == nil {
		roleSlug = string(r.Slug)
	}

	accessToken, accessExp, err := s.jwtMgr.IssueAccess(u.ID.String(), sessionID, roleSlug)
	if err != nil {
		return nil, err
	}
	refreshToken, _, err := s.jwtMgr.IssueRefresh(u.ID.String(), sessionID, roleSlug)
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
