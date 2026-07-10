package email_test

import (
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
)

func TestRenderVerifyEmail(t *testing.T) {
	r, err := email.RenderVerifyEmail(email.VerifyEmailData{
		Name:       "Alice",
		VerifyLink: "https://example.com/verify?token=abc123",
		ExpireHrs:  "24",
	})
	if err != nil {
		t.Fatalf("RenderVerifyEmail: %v", err)
	}
	assertNoPlaceholders(t, "verify-email", r)
	assertContains(t, r.HTML, "Alice", "verify link")
	assertContains(t, r.HTML, "https://example.com/verify?token=abc123", "verify link")
	assertContains(t, r.Text, "24", "expire hours")
}

func TestRenderResetPassword(t *testing.T) {
	r, err := email.RenderResetPassword(email.ResetPasswordData{
		Name:       "Bob",
		ResetLink:  "https://example.com/reset?token=xyz",
		ExpireMins: "60",
		IPAddress:  "203.0.113.1",
	})
	if err != nil {
		t.Fatalf("RenderResetPassword: %v", err)
	}
	assertNoPlaceholders(t, "reset-password", r)
	assertContains(t, r.HTML, "Bob", "name")
	assertContains(t, r.HTML, "203.0.113.1", "IP address")
}

func TestRenderOTPCode(t *testing.T) {
	r, err := email.RenderOTPCode(email.OTPCodeData{
		Name: "Carol", Code: "483920", ExpireMins: "10",
		Purpose: "Two-factor authentication",
	})
	if err != nil {
		t.Fatalf("RenderOTPCode: %v", err)
	}
	assertNoPlaceholders(t, "otp-code", r)
	assertContains(t, r.HTML, "483920", "OTP code")
	assertContains(t, r.Text, "483920", "OTP code in text")
}

func TestRenderWelcome(t *testing.T) {
	r, err := email.RenderWelcome(email.WelcomeData{
		Name:         "Dave",
		DashboardURL: "https://app.example.com",
		DocsURL:      "https://docs.example.com",
	})
	if err != nil {
		t.Fatalf("RenderWelcome: %v", err)
	}
	assertNoPlaceholders(t, "welcome", r)
	assertContains(t, r.HTML, "Dave", "name")
	assertContains(t, r.HTML, "https://app.example.com", "dashboard URL")
}

func TestRenderNotification(t *testing.T) {
	r, err := email.RenderNotification(email.NotificationData{
		Name: "Eve", Title: "Your account was accessed",
		Body:     "A new sign-in was detected.",
		CTALabel: "Review activity",
		CTAURL:   "https://app.example.com/activity",
	})
	if err != nil {
		t.Fatalf("RenderNotification: %v", err)
	}
	assertNoPlaceholders(t, "notification", r)
	assertContains(t, r.HTML, "Your account was accessed", "title")
}

func TestAllPlaceholdersReplaced(t *testing.T) {
	tests := []struct {
		name string
		fn   func() (*email.Rendered, error)
	}{
		{"verify-email", func() (*email.Rendered, error) {
			return email.RenderVerifyEmail(email.VerifyEmailData{Name: "A", VerifyLink: "https://x.com", ExpireHrs: "24"})
		}},
		{"reset-password", func() (*email.Rendered, error) {
			return email.RenderResetPassword(email.ResetPasswordData{Name: "A", ResetLink: "https://x.com", ExpireMins: "60", IPAddress: "1.2.3.4"})
		}},
		{"welcome", func() (*email.Rendered, error) {
			return email.RenderWelcome(email.WelcomeData{Name: "A", DashboardURL: "https://x.com", DocsURL: "https://d.x.com"})
		}},
		{"otp-code", func() (*email.Rendered, error) {
			return email.RenderOTPCode(email.OTPCodeData{Name: "A", Code: "123456", ExpireMins: "10", Purpose: "login"})
		}},
		{"notification", func() (*email.Rendered, error) {
			return email.RenderNotification(email.NotificationData{Name: "A", Title: "T", Body: "B", CTALabel: "Go", CTAURL: "https://x.com"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := tt.fn()
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			assertNoPlaceholders(t, tt.name, r)
		})
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func assertNoPlaceholders(t *testing.T, name string, r *email.Rendered) {
	t.Helper()
	for _, part := range []struct{ label, content string }{
		{"HTML", r.HTML}, {"Text", r.Text},
	} {
		for _, line := range strings.Split(part.content, "\n") {
			if strings.Contains(line, "__") {
				t.Errorf("%s %s: unreplaced placeholder: %s", name, part.label, strings.TrimSpace(line))
			}
		}
	}
}

func assertContains(t *testing.T, content, needle, desc string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Errorf("expected %s to contain %q", desc, needle)
	}
}
