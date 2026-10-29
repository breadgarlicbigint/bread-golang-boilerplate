package service

// Account-lifecycle actions an admin performs on someone else's login: send a
// set-password link, or (opt-in) set a password directly.
//
// The default is a LINK, not a password: the set-password email carries a
// single-use reset token, so no plaintext credential ever travels by email.
// "Email the password" is the explicit opt-in on ResetPassword (sendEmail).
//
// Two guards shape every path here:
//   - WHO may act on WHOM: only a wildcard (super admin) may target another
//     wildcard (admin) user — ErrAdminProtected otherwise.
//   - The mailer is checked BEFORE any write, so asking for an email we cannot
//     send fails cleanly instead of changing a password we then can't deliver.

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	roleentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"go.mongodb.org/mongo-driver/bson"
)

// ResetTokenIssuer mints a single-use password-reset token for a user id and
// returns it with its expiry. The auth service satisfies this; it is a port so
// the user service does not import auth.
type ResetTokenIssuer interface {
	PasswordResetToken(ctx context.Context, userID string) (token string, exp time.Time, err error)
}

// WithResetTokens attaches the token issuer used by SendSetPasswordLink.
func (s *UserService) WithResetTokens(t ResetTokenIssuer) *UserService {
	s.resetTokens = t
	return s
}

func (s *UserService) canEmail() bool { return s.localMailer != nil && s.emailQueue != nil }

// targetFor loads the user an action targets and applies the admin-protection
// guard: a non-wildcard actor may not act on a wildcard (admin) user.
func (s *UserService) targetFor(ctx context.Context, actor Actor, id string) (*entity.User, error) {
	u, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.HasWildcard {
		return u, nil
	}
	if s.roles != nil {
		if slug, err := s.roles.SlugByID(ctx, u.RoleID); err == nil && slug == string(roleentity.RoleAdmin) {
			return nil, errors.ErrAdminProtected
		}
	}
	return u, nil
}

// SendSetPasswordLink emails the target a single-use link to set their own
// password — the default way an admin hands out or resets access, with no
// credential in the message. The mailer is checked before the token is minted.
func (s *UserService) SendSetPasswordLink(ctx context.Context, actor Actor, lang, id string) error {
	if !s.canEmail() || s.resetTokens == nil {
		return errors.ErrEmailUnavailable
	}
	u, err := s.targetFor(ctx, actor, id)
	if err != nil {
		return err
	}
	token, _, err := s.resetTokens.PasswordResetToken(ctx, u.ID.String())
	if err != nil {
		return err
	}
	link := s.appURL + "/set-password?token=" + token
	rendered, err := s.localMailer.RenderResetPassword(lang, u.FirstName+" "+u.LastName, link, "60", "")
	if err != nil {
		return err
	}
	if err := s.emailQueue.EnqueueEmail(u.Email, s.localMailer.SubjectResetPassword(lang), rendered.HTML, rendered.Text); err != nil {
		return errors.ErrEmailQueueFailed
	}
	return nil
}

// ResetPassword sets the target's password to newPassword (the opt-in direct
// path — e.g. an admin reading a temporary one out). When sendEmail is true the
// mailer is checked BEFORE the write, so a password is never changed to one that
// cannot then be delivered.
func (s *UserService) ResetPassword(ctx context.Context, actor Actor, lang, id, newPassword string, sendEmail bool) error {
	if sendEmail && !s.canEmail() {
		return errors.ErrEmailUnavailable
	}
	u, err := s.targetFor(ctx, actor, id)
	if err != nil {
		return err
	}
	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.Update(ctx, u.ID, bson.M{
		"passwordHash": hash, "updatedAt": time.Now(), "passwordAttempts": 0, "lockedUntil": nil,
	}); err != nil {
		return err
	}
	if sendEmail {
		rendered, err := s.localMailer.RenderNotification(lang, u.FirstName,
			s.localMailer.SubjectResetPassword(lang),
			"Your password has been changed by an administrator.", "", "")
		if err == nil {
			if qErr := s.emailQueue.EnqueueEmail(u.Email, s.localMailer.SubjectResetPassword(lang), rendered.HTML, rendered.Text); qErr != nil {
				return errors.ErrEmailQueueFailed
			}
		}
	}
	return nil
}
