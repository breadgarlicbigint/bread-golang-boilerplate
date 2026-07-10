package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"go.mongodb.org/mongo-driver/bson"
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

type UserService struct {
	repo   UserRepo
	hasher *hash.Hasher
	cfg    config.AuthConfig
}

func New(repo UserRepo, hasher *hash.Hasher, cfg config.AuthConfig) *UserService {
	return &UserService{repo: repo, hasher: hasher, cfg: cfg}
}

func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest) (*entity.User, error) {
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

	u := &entity.User{
		Email:         req.Email,
		Username:      req.Username,
		PasswordHash:  passwordHash,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		PhoneNumber:   req.PhoneNumber,
		RoleID:        roleID,
		Status:        entity.UserStatusActive,
		EmailVerified: false,
		NotifSettings: entity.DefaultNotifSettings(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
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

func (s *UserService) Update(ctx context.Context, id string, req dto.UpdateUserRequest) (*entity.User, error) {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
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
