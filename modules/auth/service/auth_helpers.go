package service

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
)

// ── OAuth / passkey token issuance ────────────────────────────────────────────

// LoginWithGitHub issues a token pair after a successful GitHub OAuth callback.
func (s *AuthService) LoginWithGitHub(ctx context.Context, u *entity.User, ip string) (*dto.LoginResponse, error) {
	return s.issueTokenPair(ctx, u, ip)
}

// LoginWithPasskey issues a token pair after a successful WebAuthn ceremony.
func (s *AuthService) LoginWithPasskey(ctx context.Context, u *entity.User, ip string) (*dto.LoginResponse, error) {
	return s.issueTokenPair(ctx, u, ip)
}

// IssueTokenPairPublic exposes issueTokenPair for external callers (e.g. OAuth handlers).
func (s *AuthService) IssueTokenPairPublic(ctx context.Context, u *entity.User, ip string) (*dto.LoginResponse, error) {
	return s.issueTokenPair(ctx, u, ip)
}

// ── Password reset ────────────────────────────────────────────────────────────

// PasswordResetToken generates a short-lived opaque reset token stored in Redis.
func (s *AuthService) PasswordResetToken(ctx context.Context, userID string) (string, time.Time, error) {
	token, err := hash.RandomHex(32)
	if err != nil {
		return "", time.Time{}, err
	}
	exp := time.Now().Add(1 * time.Hour)
	if err := s.rdb.Set(ctx, resetKeyPrefix+token, userID, time.Until(exp)).Err(); err != nil {
		return "", time.Time{}, err
	}
	return token, exp, nil
}

// ValidatePasswordResetToken verifies the token and returns the associated userID.
func (s *AuthService) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	userID, err := s.rdb.Get(ctx, resetKeyPrefix+token).Result()
	if err != nil {
		return "", errors.ErrTokenExpired
	}
	_ = s.rdb.Del(ctx, resetKeyPrefix+token) // single-use
	return userID, nil
}

// ── Email verification ────────────────────────────────────────────────────────

// EmailVerifyToken generates a 24-hour email-verification token.
func (s *AuthService) EmailVerifyToken(ctx context.Context, userID string) (string, error) {
	token, err := hash.RandomHex(32)
	if err != nil {
		return "", err
	}
	return token, s.rdb.Set(ctx, verifyKeyPrefix+token, userID, 24*time.Hour).Err()
}

// ValidateEmailVerifyToken verifies the token and returns the userID.
func (s *AuthService) ValidateEmailVerifyToken(ctx context.Context, token string) (string, error) {
	userID, err := s.rdb.Get(ctx, verifyKeyPrefix+token).Result()
	if err != nil {
		return "", errors.ErrTokenExpired
	}
	_ = s.rdb.Del(ctx, verifyKeyPrefix+token)
	return userID, nil
}

// ── Email sending (React Email + i18n via LocalizedMailer) ────────────────────

// SendVerificationEmail generates a token and sends the verify-email template
// in the user's preferred language. lang comes from the x-custom-lang header.
func (s *AuthService) SendVerificationEmail(ctx context.Context, lang, userID, emailAddr, name, appURL string) error {
	if s.localMailer == nil {
		return nil // email not configured — skip silently in dev
	}
	token, err := s.EmailVerifyToken(ctx, userID)
	if err != nil {
		return err
	}
	verifyLink := appURL + "/v1/auth/verify-email?token=" + token
	return s.localMailer.SendVerifyEmail(ctx, lang, emailAddr, name, verifyLink, "24")
}

// SendPasswordResetEmail generates a reset token and sends the reset-password template.
func (s *AuthService) SendPasswordResetEmail(ctx context.Context, lang, userID, emailAddr, name, appURL, ip string) error {
	if s.localMailer == nil {
		return nil
	}
	token, _, err := s.PasswordResetToken(ctx, userID)
	if err != nil {
		return err
	}
	resetLink := appURL + "/v1/auth/reset-password?token=" + token
	return s.localMailer.SendPasswordReset(ctx, lang, emailAddr, name, resetLink, "60", ip)
}

// SendWelcomeEmail sends the welcome template after registration.
func (s *AuthService) SendWelcomeEmail(ctx context.Context, lang, emailAddr, name, appURL string) error {
	if s.localMailer == nil {
		return nil
	}
	return s.localMailer.SendWelcome(ctx, lang, emailAddr, name, appURL+"/dashboard", appURL+"/docs")
}
