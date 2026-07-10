# Multi-Language Email System

All transactional emails are rendered with **React Email** and fully localised
via the Go i18n system. No visible string is hardcoded in any template.

---

## Architecture

```
email-templates/src/emails/*.tsx   ← React Email components (structure + style only)
         │  make build-emails (npm run build)
         ▼
pkg/email/dist/*.html + *.txt      ← Compiled HTML with __TOKEN__ placeholders
         │  //go:embed
         ▼
pkg/email/localized_mailer.go      ← Resolves all tokens from locales/*.json
         │
         ▼
locales/en.json  locales/id.json   ← Translated strings for every token
```

---

## How tokens work

React Email handles layout, CSS inlining, and HTML email compliance.
Every visible string is a `__TOKEN__` constant:

```tsx
// verify-email.tsx
const P = {
  heading:    "__HEADING__",       // ← text token  (i18n)
  greeting:   "__GREETING__",      // ← text token  (i18n, includes {name})
  verifyLink: "__VERIFY_LINK__",   // ← data token  (caller-supplied)
  expireHrs:  "__EXPIRE_HRS__",    // ← data token  (caller-supplied)
  ...
};
```

Go resolves every token before the HTML is sent:

```go
// LocalizedMailer.RenderVerifyEmail("id", "Budi", link, "24")
// Produces:
// __HEADING__  → "Verifikasi email Anda"      (from locales/id.json)
// __GREETING__ → "Halo Budi, terima kasih!"  (i18n with {name} substituted)
// __VERIFY_LINK__ → "https://app.example.com/verify?token=abc"
// __EXPIRE_HRS__  → "24"
```

---

## Adding a new language

1. Create `locales/fr.json` (copy from `en.json`, translate all `email.*` keys):

```json
{
  "email": {
    "layout": {
      "brandName":       "Bread Boilerplate",
      "footerCopyright": "© {year} Bread Boilerplate. Tous droits réservés.",
      "footerIgnore":    "Si vous n'avez pas demandé cet e-mail, ignorez-le."
    },
    "verifyEmail": {
      "subject":    "Vérifiez votre adresse e-mail",
      "heading":    "Vérifiez votre e-mail",
      "greeting":   "Bonjour {name}, merci de vous être inscrit !",
      ...
    }
  }
}
```

2. Place it in `locales/fr.json` — the i18n system picks it up automatically.

3. No rebuild needed. Templates are already compiled; only locale data changes.

---

## Supported templates

| Template | Tokens | Send method |
|---|---|---|
| `verify-email` | name, verifyLink, expireHrs + 8 text tokens | `localMailer.SendVerifyEmail(ctx, lang, to, ...)` |
| `reset-password` | name, resetLink, expireMins, ip + 9 text tokens | `localMailer.SendPasswordReset(ctx, lang, to, ...)` |
| `welcome` | name, dashboardUrl, docsUrl + 12 text tokens (incl. 3 features) | `localMailer.SendWelcome(ctx, lang, to, ...)` |
| `otp-code` | name, code, expireMins, purpose + 6 text tokens | `localMailer.SendOTPCode(ctx, lang, to, ...)` |
| `notification` | name, title, body, ctaLabel, ctaUrl + 2 text tokens | `localMailer.SendNotification(ctx, lang, to, ...)` |

---

## How language is determined

Language is read from the `x-custom-lang` request header (set by the
`pkgi18n.Middleware` in `app.go`). Handlers extract it with:

```go
_, lang := pkgi18n.FromContext(c)
```

Pass `lang` to auth service email helpers:

```go
// in auth handler — after registration:
_ = s.authSvc.SendWelcomeEmail(ctx, lang, user.Email, user.FirstName, appURL)

// forgot password:
_ = s.authSvc.SendPasswordResetEmail(ctx, lang, user.ID.Hex(),
    user.Email, user.FirstName, appURL, c.ClientIP())
```

If the requested language is not found, the system falls back to **English** automatically.

---

## Building the templates

```bash
# First-time or after changing .tsx files:
make build-emails

# What it does:
cd email-templates && npm ci && npm run build
# → compiles React Email .tsx → HTML/text → pkg/email/dist/
# → these files are embedded in the Go binary via //go:embed
```

Docker handles this automatically in the `email-builder` multi-stage build.

---

## Locale key structure

```
locales/
├── en.json     English (default — fallback for any unknown language)
└── id.json     Indonesian (Bahasa Indonesia)

email.*
├── layout.*
│   ├── brandName
│   ├── footerCopyright   (supports {year})
│   └── footerIgnore
├── verifyEmail.*
│   ├── subject           (no interpolation)
│   ├── preview           (supports {name})
│   ├── heading
│   ├── greeting          (supports {name})
│   ├── body
│   ├── ctaLabel
│   ├── copyLinkText
│   ├── expireNote        (supports {expireHrs})
│   └── ignoreNote
├── resetPassword.*
│   ├── subject, preview, heading, greeting, body
│   ├── ctaLabel, copyLinkText
│   ├── securityTitle
│   ├── securityNote      (supports {expireMins}, {ipAddress})
│   └── ignoreNote
├── welcome.*
│   ├── subject, preview, heading, greeting, subheading
│   ├── ctaPrimary, ctaSecondary, helpNote
│   └── feat1Icon/Title/Desc, feat2*, feat3*
├── otpCode.*
│   ├── subject, preview  (supports {purpose}, {code})
│   ├── heading
│   ├── greeting          (supports {name}, {purpose})
│   ├── expireNote        (supports {expireMins})
│   ├── warningTitle
│   └── warningBody
└── notification.*
    └── greeting          (supports {name})
```

---

## Testing

```bash
# Unit tests — no SES connection required
go test ./pkg/email/... -v -run TestLocalized

# Test a specific language
go test ./pkg/email/... -v -run TestLocalizedMailer_VerifyEmail_Indonesian
```

The `LocalizedMailer.Render*` methods work with a `nil` Mailer (no AWS
credentials needed) so the full render pipeline can be tested in CI.

---

## Preview emails locally

```bash
cd email-templates
npm run preview
# Opens React Email dev server at http://localhost:3333
# Live-reload as you edit .tsx files
# Note: preview shows placeholder tokens, not translated values
```
