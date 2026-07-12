package email_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	pkgi18n "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
)

// unreplacedTokenRE matches our __TOKEN__ convention (uppercase, digits,
// underscores only) — it must not match unrelated library-internal markup
// such as @react-email/components' `data-id="__react-email-column"`.
var unreplacedTokenRE = regexp.MustCompile(`__[A-Z0-9_]+__`)

// findLocalesDir walks up from the test file to locate the locales directory.
func findLocalesDir(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for {
		candidate := filepath.Join(dir, "locales")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("locales/ directory not found — skipping i18n tests")
	return ""
}

func newTestTranslator(t *testing.T) *pkgi18n.Translator {
	t.Helper()
	localesDir := findLocalesDir(t)
	tr, err := pkgi18n.New(localesDir)
	if err != nil {
		t.Fatalf("i18n.New: %v", err)
	}
	return tr
}

func newTestLocalizedMailer(t *testing.T) *email.LocalizedMailer {
	t.Helper()
	tr := newTestTranslator(t)
	// Mailer is nil — LocalizedMailer.Send won't be called in unit tests
	// (we only test token resolution + rendering, not actual SES dispatch)
	return email.NewLocalizedMailer(nil, tr)
}

// ── Render-only helpers (bypass Send) ────────────────────────────────────────

func TestLocalizedMailer_VerifyEmail_English(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderVerifyEmail("en", "Alice", "https://example.com/verify?t=abc", "24")
	if err != nil {
		t.Fatalf("RenderVerifyEmail: %v", err)
	}

	assertNoUnreplacedTokens(t, "en verify-email HTML", r.HTML)
	assertNoUnreplacedTokens(t, "en verify-email Text", r.Text)
	assertContainsStr(t, r.HTML, "Alice", "name")
	assertContainsStr(t, r.HTML, "Verify your email", "heading")
	assertContainsStr(t, r.HTML, "https://example.com/verify?t=abc", "link")
	assertContainsStr(t, r.HTML, "24", "expire hours")
	t.Logf("subject: %s", m.SubjectVerifyEmail("en", "Alice"))
}

func TestLocalizedMailer_VerifyEmail_Indonesian(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderVerifyEmail("id", "Budi", "https://example.com/verify?t=xyz", "24")
	if err != nil {
		t.Fatalf("RenderVerifyEmail (id): %v", err)
	}

	assertNoUnreplacedTokens(t, "id verify-email HTML", r.HTML)
	assertContainsStr(t, r.HTML, "Budi", "name")
	assertContainsStr(t, r.HTML, "Verifikasi email Anda", "Indonesian heading")
	assertContainsStr(t, r.HTML, "Verifikasi Alamat Email", "Indonesian CTA")
	t.Logf("subject (id): %s", m.SubjectVerifyEmail("id", "Budi"))
}

func TestLocalizedMailer_ResetPassword_English(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderResetPassword("en", "Bob", "https://example.com/reset?t=abc", "60", "203.0.113.1")
	if err != nil {
		t.Fatalf("RenderResetPassword: %v", err)
	}

	assertNoUnreplacedTokens(t, "en reset-password HTML", r.HTML)
	assertContainsStr(t, r.HTML, "Bob", "name")
	assertContainsStr(t, r.HTML, "203.0.113.1", "IP address")
	assertContainsStr(t, r.HTML, "60", "expiry minutes")
	assertContainsStr(t, r.HTML, "Reset your password", "heading")
}

func TestLocalizedMailer_ResetPassword_Indonesian(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderResetPassword("id", "Siti", "https://example.com/reset?t=xyz", "60", "1.2.3.4")
	if err != nil {
		t.Fatalf("RenderResetPassword (id): %v", err)
	}

	assertNoUnreplacedTokens(t, "id reset-password HTML", r.HTML)
	assertContainsStr(t, r.HTML, "Atur ulang kata sandi", "Indonesian heading")
}

func TestLocalizedMailer_OTPCode_English(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderOTPCode("en", "Carol", "483920", "10", "two-factor authentication")
	if err != nil {
		t.Fatalf("RenderOTPCode: %v", err)
	}

	assertNoUnreplacedTokens(t, "en otp HTML", r.HTML)
	assertContainsStr(t, r.HTML, "483920", "OTP code")
	assertContainsStr(t, r.HTML, "10", "expire minutes")
	assertContainsStr(t, r.HTML, "Never share", "warning")
}

func TestLocalizedMailer_OTPCode_Indonesian(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderOTPCode("id", "Dewi", "192837", "10", "autentikasi dua faktor")
	if err != nil {
		t.Fatalf("RenderOTPCode (id): %v", err)
	}

	assertNoUnreplacedTokens(t, "id otp HTML", r.HTML)
	assertContainsStr(t, r.HTML, "192837", "code")
	assertContainsStr(t, r.HTML, "Jangan bagikan", "Indonesian warning")
}

func TestLocalizedMailer_Welcome_English(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderWelcome("en", "Dave", "https://app.example.com", "https://docs.example.com")
	if err != nil {
		t.Fatalf("RenderWelcome: %v", err)
	}

	assertNoUnreplacedTokens(t, "en welcome HTML", r.HTML)
	assertContainsStr(t, r.HTML, "Dave", "name")
	assertContainsStr(t, r.HTML, "Welcome aboard", "heading")
	assertContainsStr(t, r.HTML, "Secure by default", "feature 1")
	assertContainsStr(t, r.HTML, "https://app.example.com", "dashboard URL")
}

func TestLocalizedMailer_Welcome_Indonesian(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderWelcome("id", "Eka", "https://app.example.com", "https://docs.example.com")
	if err != nil {
		t.Fatalf("RenderWelcome (id): %v", err)
	}

	assertNoUnreplacedTokens(t, "id welcome HTML", r.HTML)
	assertContainsStr(t, r.HTML, "Selamat bergabung", "Indonesian heading")
	assertContainsStr(t, r.HTML, "Aman secara bawaan", "Indonesian feature 1")
}

func TestLocalizedMailer_Notification_English(t *testing.T) {
	m := newTestLocalizedMailer(t)
	r, err := m.RenderNotification("en", "Eve",
		"New login detected", "A new sign-in was detected from Chrome on Mac.",
		"Review activity", "https://app.example.com/activity")
	if err != nil {
		t.Fatalf("RenderNotification: %v", err)
	}

	assertNoUnreplacedTokens(t, "en notification HTML", r.HTML)
	assertContainsStr(t, r.HTML, "Eve", "name")
	assertContainsStr(t, r.HTML, "New login detected", "title")
}

func TestLocalizedMailer_FallbackToEnglish(t *testing.T) {
	m := newTestLocalizedMailer(t)
	// "fr" is not a known locale — should fall back to "en"
	r, err := m.RenderVerifyEmail("fr", "Marie", "https://example.com/verify", "24")
	if err != nil {
		t.Fatalf("RenderVerifyEmail (fr fallback): %v", err)
	}
	assertNoUnreplacedTokens(t, "fr fallback HTML", r.HTML)
	// Should still have English text (fallback)
	assertContainsStr(t, r.HTML, "Verify your email", "English fallback heading")
}

func TestLocalizedMailer_AllLanguages_NoUnreplacedTokens(t *testing.T) {
	m := newTestLocalizedMailer(t)
	langs := []string{"en", "id"}

	tests := []struct {
		name string
		fn   func(lang string) (*email.Rendered, error)
	}{
		{"verify-email", func(l string) (*email.Rendered, error) {
			return m.RenderVerifyEmail(l, "Test", "https://x.com/v", "24")
		}},
		{"reset-password", func(l string) (*email.Rendered, error) {
			return m.RenderResetPassword(l, "Test", "https://x.com/r", "60", "1.2.3.4")
		}},
		{"welcome", func(l string) (*email.Rendered, error) {
			return m.RenderWelcome(l, "Test", "https://app.x.com", "https://docs.x.com")
		}},
		{"otp-code", func(l string) (*email.Rendered, error) {
			return m.RenderOTPCode(l, "Test", "123456", "10", "login")
		}},
		{"notification", func(l string) (*email.Rendered, error) {
			return m.RenderNotification(l, "Test", "Title", "Body", "Go", "https://x.com")
		}},
	}

	for _, lang := range langs {
		for _, tt := range tests {
			t.Run(lang+"/"+tt.name, func(t *testing.T) {
				r, err := tt.fn(lang)
				if err != nil {
					t.Fatalf("render: %v", err)
				}
				assertNoUnreplacedTokens(t, lang+"/"+tt.name+" HTML", r.HTML)
				assertNoUnreplacedTokens(t, lang+"/"+tt.name+" Text", r.Text)
			})
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func assertNoUnreplacedTokens(t *testing.T, label, content string) {
	t.Helper()
	for _, line := range strings.Split(content, "\n") {
		if unreplacedTokenRE.MatchString(line) {
			t.Errorf("%s: unreplaced token in: %s", label, strings.TrimSpace(line))
		}
	}
}

func assertContainsStr(t *testing.T, content, needle, desc string) {
	t.Helper()
	if !strings.Contains(content, needle) {
		t.Errorf("expected %s to contain %q", desc, needle)
	}
}
