package email_test

import (
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
)

// Every user-authored value interpolated into an HTML email body must be
// html.EscapeString()ed — a name, a notification title/body or any other value
// carrying `<`, `&` or `"` would otherwise break the markup or inject it — while
// the plain-text body keeps the value raw. One case per template, since each has
// its own Render method and its own set of user-supplied tokens.

const marker = `X<Y>&"Z`           // the raw user value
const escapedMarker = "X&lt;Y&gt;&amp;&#34;Z" // what it must look like in HTML

func assertEscapedInHTMLRawInText(t *testing.T, r *email.Rendered, where string) {
	t.Helper()
	if !strings.Contains(r.HTML, escapedMarker) {
		t.Errorf("%s: HTML does not carry the escaped value %q", where, escapedMarker)
	}
	if strings.Contains(r.HTML, marker) {
		t.Errorf("%s: HTML carries the raw (unescaped) value %q — markup can be injected", where, marker)
	}
	if !strings.Contains(r.Text, marker) {
		t.Errorf("%s: text body does not carry the raw value %q", where, marker)
	}
	if strings.Contains(r.Text, escapedMarker) {
		t.Errorf("%s: text body carries HTML-escaped entities %q", where, escapedMarker)
	}
}

func TestEmailEscaping_PerTemplate(t *testing.T) {
	m := newTestLocalizedMailer(t)

	t.Run("verify-email/name", func(t *testing.T) {
		r, err := m.RenderVerifyEmail("en", marker, "https://x/v", "24")
		if err != nil {
			t.Fatal(err)
		}
		assertEscapedInHTMLRawInText(t, r, "verify-email")
	})

	t.Run("reset-password/name", func(t *testing.T) {
		r, err := m.RenderResetPassword("en", marker, "https://x/r", "60", "1.2.3.4")
		if err != nil {
			t.Fatal(err)
		}
		assertEscapedInHTMLRawInText(t, r, "reset-password")
	})

	t.Run("welcome/name", func(t *testing.T) {
		r, err := m.RenderWelcome("en", marker, "https://app/x", "https://docs/x")
		if err != nil {
			t.Fatal(err)
		}
		assertEscapedInHTMLRawInText(t, r, "welcome")
	})

	t.Run("otp-code/name", func(t *testing.T) {
		r, err := m.RenderOTPCode("en", marker, "123456", "10", "login")
		if err != nil {
			t.Fatal(err)
		}
		assertEscapedInHTMLRawInText(t, r, "otp-code")
	})

	// The notification title/body are admin-authored free text — the clearest
	// injection vector of the set.
	t.Run("notification/body", func(t *testing.T) {
		r, err := m.RenderNotification("en", "Eve", "A title", marker, "Go", "https://x/a")
		if err != nil {
			t.Fatal(err)
		}
		assertEscapedInHTMLRawInText(t, r, "notification")
	})
}
