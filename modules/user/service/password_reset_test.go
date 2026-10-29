package service

import (
	"context"
	"testing"

	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
)

// The mailer is checked BEFORE anything is read or written: an action that must
// email a link cannot proceed with no mailer, and must not half-happen. With no
// mailer/queue/token issuer wired, SendSetPasswordLink fails closed up front.
func TestSendSetPasswordLink_FailsClosedWithoutMailer(t *testing.T) {
	s := &UserService{} // no localMailer, no emailQueue, no resetTokens
	if err := s.SendSetPasswordLink(context.Background(), Actor{HasWildcard: true}, "en", "u1"); err != apperrors.ErrEmailUnavailable {
		t.Fatalf("err = %v, want ErrEmailUnavailable", err)
	}
}

// ResetPassword with sendEmail=true also checks the mailer before the write, so
// a password is never changed to one that cannot then be delivered.
func TestResetPassword_SendEmailChecksMailerFirst(t *testing.T) {
	s := &UserService{}
	if err := s.ResetPassword(context.Background(), Actor{HasWildcard: true}, "en", "u1", "newpass123", true); err != apperrors.ErrEmailUnavailable {
		t.Fatalf("err = %v, want ErrEmailUnavailable", err)
	}
}
