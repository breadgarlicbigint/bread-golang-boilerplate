package dto

import (
	"time"

	"github.com/google/uuid"
)

// ── Request DTOs ──────────────────────────────────────────────────────────────

type CreateUserRequest struct {
	Email       string `json:"email"       validate:"required,email"`
	Username    string `json:"username"    validate:"required,min=3,max=30,alphanum"`
	Password    string `json:"password"    validate:"required,min=8"`
	FirstName   string `json:"firstName"   validate:"required,min=1,max=50"`
	LastName    string `json:"lastName"    validate:"required,min=1,max=50"`
	PhoneNumber string `json:"phoneNumber" validate:"omitempty,e164"`
	RoleID      string `json:"roleId"      validate:"required"`
}

type UpdateUserRequest struct {
	FirstName      string `json:"firstName"      validate:"omitempty,min=1,max=50"`
	LastName       string `json:"lastName"       validate:"omitempty,min=1,max=50"`
	PhoneNumber    string `json:"phoneNumber"    validate:"omitempty,e164"`
	ProfilePicture string `json:"profilePicture" validate:"omitempty,url"`
	Gender         string `json:"gender"         validate:"omitempty,oneof=male female other"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

type UpdateNotifSettingsRequest struct {
	EmailEnabled bool `json:"emailEnabled"`
	PushEnabled  bool `json:"pushEnabled"`
	InAppEnabled bool `json:"inAppEnabled"`
}

type BlockUserRequest struct {
	Reason string `json:"reason" validate:"required,min=5"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type UserResponse struct {
	ID             uuid.UUID `json:"id"`
	Email          string             `json:"email"`
	Username       string             `json:"username"`
	FirstName      string             `json:"firstName"`
	LastName       string             `json:"lastName"`
	PhoneNumber    string             `json:"phoneNumber,omitempty"`
	ProfilePicture string             `json:"profilePicture,omitempty"`
	Gender         string             `json:"gender,omitempty"`
	Status         string             `json:"status"`
	RoleID         uuid.UUID `json:"roleId"`
	EmailVerified  bool               `json:"emailVerified"`
	TwoFAEnabled   bool               `json:"twoFAEnabled"`
	LastLoginAt    *time.Time         `json:"lastLoginAt,omitempty"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
}

type UserListResponse struct {
	Users []UserResponse `json:"users"`
}
