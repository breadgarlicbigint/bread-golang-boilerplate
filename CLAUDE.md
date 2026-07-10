# CLAUDE.md — Claude Code Project Instructions

This file tells Claude Code everything it needs to know to work effectively on
this codebase. Read the entire file before making changes.

---

## Project Overview

**Bread Golang Boilerplate** — production-ready Go REST API boilerplate, a full
port of [ack-nestjs-boilerplate](https://github.com/andrechristikan/ack-nestjs-boilerplate)
to idiomatic Go.

**Module:** `github.com/breadgarlicbigint/bread-golang-boilerplate`
**Go version:** 1.23
**Node.js version:** 20+ (for `make build-emails` and the `web/` test client — never in production)

---

## Quick Commands

```bash
make setup              # first-time: keys + .env + build-emails + go mod tidy + swagger
make docker-up          # start MongoDB rs + Redis + API
make seed               # populate DB (runs inside Docker network — requires docker-up first)
make seed-local         # seed against localhost (reads .env via BREAD_CONFIG_FILE)
make dev                # hot-reload with air
make test               # all unit tests
make test-coverage      # HTML coverage report → coverage.html
make swagger            # regenerate Swagger docs (uses go run, no install needed)
make build-emails       # compile React Email .tsx → pkg/email/dist/ (requires Node.js)
make web-install        # install web/ (React test client) dependencies
make web-dev            # run the web test client → http://localhost:5173
make lint               # golangci-lint
make help               # full target list
```

---

## Tech Stack

| Layer | Library |
|---|---|
| HTTP | `gin-gonic/gin` |
| Database | `go.mongodb.org/mongo-driver` (MongoDB 8, replica set required) |
| Cache / Sessions | `redis/go-redis/v9` |
| JWT | `golang-jwt/jwt/v5` — ES256 access (15 min) + ES512 refresh (7 days) |
| Password | `golang.org/x/crypto/bcrypt` (cost 12) |
| 2FA | `pquerna/otp` (TOTP) |
| Passkey/WebAuthn | `go-webauthn/webauthn` |
| Background jobs | `hibiken/asynq` (Redis-backed, mirrors BullMQ) |
| Config | `spf13/viper` (.env + env vars + defaults) |
| Validation | `go-playground/validator/v10` (centralised in `shared/validate`) |
| Logging | `go.uber.org/zap` |
| Swagger | `swaggo/swag` + `swaggo/gin-swagger` |
| Email templates | React Email (build-time, Node.js) → embedded HTML (`//go:embed`) |
| Email delivery | AWS SES or SMTP — selected via `MAIL_DRIVER` (`pkg/email`) |
| SMS / WhatsApp | `twilio/twilio-go` |
| Push notifications | Firebase FCM (`firebase.google.com/go/v4`) |
| Storage | AWS S3 (`aws-sdk-go-v2/service/s3`) |
| OAuth | `golang.org/x/oauth2` (Google, GitHub) |
| i18n | Custom `pkg/i18n` — reads `locales/*.json`, `x-custom-lang` header |
| Error tracking | `getsentry/sentry-go` |
| Rate limiting | Redis sliding-window in `shared/middleware/ratelimit.go` |

---

## Architecture

### Monorepo structure

```
apps/                           ← Deployable binaries
  api/
    main.go                     ← HTTP server entry point
    swagger.go                  ← blank import registers swagger docs
    app/app.go                  ← dependency wiring (all modules assembled here)
  worker/
    main.go                     ← Asynq background worker entry point

modules/                        ← Domain modules (future microservices)
  auth/
    contract.go                 ← PUBLIC interface — what other modules may call
    entity/ dto/ repository/ service/ handler/
  user/            ← same layout
  role/
  apikey/
  activity/
  analytics/
  appversion/
  featureflag/
  health/
  mobile/
  notification/
  passkey/
  tenant/

shared/                         ← Cross-cutting infrastructure (no business logic)
  config/                       ← Viper config loader (all env vars)
  database/                     ← MongoDB + Redis clients
  errors/                       ← Domain error types + sentinels
  middleware/                   ← HTTP middleware (auth, rate-limit, security, i18n…)
  pagination/                   ← Query binding + Skip/Limit/BuildMeta helpers
  response/                     ← Standard JSON envelope helpers
  validate/                     ← validate.BindJSON / validate.BindQuery

pkg/                            ← Pure utilities (zero domain knowledge)
  dbid/                         ← ID strategy: uuid or objectid — see docs/id-migration.md
  email/                        ← SES mailer + React Email renderer + LocalizedMailer
  hash/                         ← bcrypt wrapper
  i18n/                         ← Locale loader + Translator + Gin middleware
  jwt/                          ← ES256/ES512 key manager
  logger/                       ← Zap factory
  sms/                          ← Twilio SMS + WhatsApp sender
  storage/                      ← AWS S3 presign / upload / delete
  uuidbson/                     ← UUID BSON Binary-4 codec
  worker/                       ← Asynq client + server + task handlers

locales/                        ← en.json, id.json (i18n strings incl. email tokens)
email-templates/                ← React Email .tsx source (build-time, Node.js only)
web/                             ← React + TS test client — see "Web Test Client" below
scripts/seed/ scripts/migrate/  ← One-shot DB seed and index creation
docs/                           ← Architecture docs, id-migration, email-i18n, swagger
```

### Per-module layout (consistent across all modules)

```
modules/<module>/
  contract.go  ← PUBLIC interface — the only thing other modules may import.
  │             When extracting to a microservice, swap concrete impl with
  │             an HTTP/gRPC client that satisfies this interface.
  entity/      ← MongoDB document struct (BSON tags, no business logic)
  dto/         ← Request/response shapes (validate tags, no DB types)
  repository/  ← Data access only (mongo driver calls, no business logic)
  service/     ← Business logic (calls repository, returns domain errors)
  handler/     ← HTTP: binds request → calls service → writes response
```

**Cross-module communication rule:** modules NEVER import each other's
`service/`, `repository/`, or `entity/` packages directly. They interact
only through the target module's `contract.go` interface. This enforces the
boundary that makes microservice extraction painless.

```go
// ✅ CORRECT — import the contract, not the internals
import auth "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth"
var _ auth.Service = (*authsvc.AuthService)(nil) // verify concrete type satisfies contract

// ❌ WRONG — coupling to internals breaks microservice extraction
import authsvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/service"
```

### Microservice extraction checklist (per module)

When a module grows large enough to extract:

1. Copy `modules/<module>/` to a new repo
2. Update its `go.mod` with its own module path
3. Replace the `contract.go` interface with an HTTP/gRPC client in the monorepo
4. The monorepo's `apps/api/app/app.go` wiring changes from `authsvc.New(...)` to `authclient.New(addr)`
5. All other modules see no change — they only used the contract interface

---

## Coding Conventions

### Handlers — always use validate.BindJSON

```go
// ✅ CORRECT — always do this
var req dto.LoginRequest
if !validate.BindJSON(c, &req) {
    return  // response already written
}

// ❌ NEVER do this
if err := c.ShouldBindJSON(&req); err != nil {
    response.BadRequest(c, err.Error())
    return
}
// ❌ NEVER do this either
if err := h.validate.Struct(req); err != nil {
    // ...
}
```

`validate.BindJSON` always produces consistent 422 responses with `errors[]`
field details using JSON tag names (not Go struct names) and human-readable messages.

### Error handling in handlers

```go
func handleError(c *gin.Context, err error) {
    if ae, ok := errors.As(err); ok {
        response.Error(c, ae.Status, ae.Message)
        return
    }
    response.InternalServerError(c, "An unexpected error occurred")
}
```

Never expose raw error strings to the client. All domain errors go through
`shared/errors.AppError` which carries a status code and safe message.

### Response envelope

All endpoints return the standard envelope:

```json
{
  "statusCode": 200,
  "message": "User fetched",
  "data": { ... },
  "_metadata": { "total": 100, "page": 1, "perPage": 20, ... },
  "timestamp": "2026-01-01T00:00:00Z",
  "path": "/v1/users",
  "requestId": "uuid"
}
```

Use `response.OK`, `response.Created`, `response.OKWithMeta`, `response.Error`, etc.
For i18n-aware responses use `response.OKI18n`, `response.ErrorI18n`.

### UUID Primary Keys

All MongoDB `_id` fields and foreign-key references use `uuid.UUID` (BSON Binary subtype 4),
not `primitive.ObjectID`. The UUID codec is registered automatically in `database/mongo.go`.

```go
// Entity struct
type User struct {
    ID     uuid.UUID `bson:"_id"    json:"id"`
    RoleID uuid.UUID `bson:"roleId" json:"roleId"`
    ...
}

// Create a new document
u := &entity.User{
    ID:     uuid.New(),          // ✅ use uuid.New(), NOT primitive.NewObjectID()
    ...
}

// Parse from URL param or JWT claim
uid, err := uuid.Parse(c.Param("id"))   // ✅ NOT primitive.ObjectIDFromHex()
if err != nil {
    response.BadRequest(c, "Invalid ID format")
    return
}

// Use in bson filter — UUID codec encodes it as Binary subtype 4 automatically
filter := bson.M{"_id": uid}

// JSON output — uuid.UUID marshals to string "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

The codec lives in `pkg/uuidbson/`. Never import `go.mongodb.org/mongo-driver/bson/primitive`
in entity, repository, service, or handler files. `primitive` has no place outside `pkg/uuidbson/`.

**`uuid.UUID` method reference — common mistakes:**

```go
// ✅ CORRECT
uid.String()            // "550e8400-e29b-41d4-a716-446655440000"
uuid.New()              // generate new UUID
uuid.Parse(someString)  // parse from string, returns (uuid.UUID, error)

// ❌ WRONG — these are primitive.ObjectID methods, uuid.UUID does NOT have them
uid.Hex()               // compile error: uuid.UUID has no method Hex
primitive.NewObjectID() // wrong type
primitive.ObjectIDFromHex(s) // wrong type
```

### Validation DTOs

Always add `validate:` tags. Use JSON tag names (lowercase):

```go
type CreateUserRequest struct {
    Email    string `json:"email"    validate:"required,email"`
    Username string `json:"username" validate:"required,min=3,max=30,alphanum"`
    Password string `json:"password" validate:"required,min=8"`
    Role     string `json:"role"     validate:"required,oneof=admin user member"`
    Phone    string `json:"phone"    validate:"omitempty,e164"`
}
```

### Service interfaces

Services expose interfaces, not concrete types. Handlers depend on interfaces:

```go
// In handler file:
type UserSvc interface {
    Create(ctx context.Context, req dto.CreateUserRequest) (*entity.User, error)
    GetByID(ctx context.Context, id string) (*entity.User, error)
    // ...
}

type UserHandler struct{ svc UserSvc }
```

### MongoDB repositories

- All collections are fetched from `database.MongoDB.Collection(name)`
- Use `bson.M` for filters, `bson.D` for ordered documents (sort, index keys)
- Soft delete: set `deletedAt` field, never physically delete users
- Always filter `deletedAt: nil` in queries that should exclude deleted docs
- Use `primitive.NewObjectID()` for new documents

### i18n

Language comes from the `x-custom-lang` request header (set by `pkgi18n.Middleware`):

```go
// In a handler:
_, lang := pkgi18n.FromContext(c)

// Translate a key:
msg := pkgi18n.TC(c, "auth.loginSuccess")

// Pass lang to service for email sending:
s.authSvc.SendWelcomeEmail(ctx, lang, email, name, appURL)
```

All visible strings in API responses should use i18n keys. Add new keys to
both `locales/en.json` and `locales/id.json`.

### Bidirectional Activity Logging

Every HTTP request auto-generates two linked log entries via `ActivityLogger`
middleware. For specific user actions log manually:

```go
actSvc.LogUserAction(ctx, correlationID, actorID, action, resourceID,
    resource, tenantID, ip, ua, metadata)

// For outbound side-effects:
actSvc.LogEmailSent(ctx, correlationID, actorID, recipient, subject, success, errMsg, latencyMS)
actSvc.LogSMSSent(ctx, correlationID, actorID, phone, success, errMsg)
```

Use the action constants in `activity/service/activity.service.go`:
`ActionUserLoginCredential`, `ActionUserRegister`, `ActionPasskeyRegistered`, etc.

---

## Adding a New Module

1. Create directory tree:
   ```
   modules/<module>/
     contract.go      ← define the public Service interface first
     entity/
     dto/
     repository/
     service/
     handler/
   ```

2. **Entity** — MongoDB struct with BSON tags
3. **DTO** — request/response shapes with `validate:` tags
4. **Repository** — data access, inject `*database.MongoDB`
5. **Service** — business logic, accepts repository interface
6. **Handler** — inject service interface, use `validate.BindJSON`, call `response.*`
7. **Register routes** in `apps/api/app/app.go`
8. **Add MongoDB indexes** in `scripts/migrate/main.go`
9. **Add Swagger annotations** (`@Summary`, `@Tags`, `@Router`, `@Param`, `@Success`, `@Failure`)
10. Run `make swagger` to regenerate docs

---

## Adding a New API Endpoint

```go
// 1. Add DTO fields with validate tags (dto/xxx.dto.go)
type CreateXxxRequest struct {
    Name string `json:"name" validate:"required,min=2,max=100"`
}

// 2. Add service method (service/xxx.service.go)
func (s *XxxService) Create(ctx context.Context, req dto.CreateXxxRequest) (*entity.Xxx, error) {
    // business logic
}

// 3. Add handler method with Swagger annotations (handler/xxx.handler.go)
// @Summary     Create xxx
// @Tags        xxx
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body dto.CreateXxxRequest true "Payload"
// @Success     201 {object} dto.XxxResponse
// @Failure     422 {object} response.ErrorEnvelope
// @Router      /v1/xxx [post]
func (h *XxxHandler) Create(c *gin.Context) {
    var req dto.CreateXxxRequest
    if !validate.BindJSON(c, &req) {
        return
    }
    result, err := h.svc.Create(c.Request.Context(), req)
    if err != nil {
        handleError(c, err)
        return
    }
    response.Created(c, "Xxx created", mapToResponse(result))
}
```

---

## Email System

### How it works

```
email-templates/src/emails/*.tsx   (React Email — build-time only)
         │  make build-emails
         ▼
pkg/email/dist/*.html + *.txt      (compiled + embedded via //go:embed)
         │
pkg/email/localized_mailer.go      (resolves __TOKEN__ values from locales/*.json)
         │
pkg/email/mailer.go                (Mailer — transport-agnostic, holds a Sender)
         │
pkg/email/factory.go               (NewMailerFromConfig — picks the Sender via MAIL_DRIVER)
         │
    ┌────┴────┐
pkg/email/ses.go   pkg/email/smtp.go   (SESSender / SMTPSender — both implement Sender)
```

### Choosing a transport: SES vs SMTP

`MAIL_DRIVER` selects the transport (`.env`):

```env
MAIL_DRIVER=ses     # default — needs AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY/AWS_SES_*
# or
MAIL_DRIVER=smtp    # needs SMTP_HOST/SMTP_PORT/SMTP_USERNAME/SMTP_PASSWORD/SMTP_FROM_*
```

`email.NewMailerFromConfig(cfg, log)` (called once in `apps/api/app/app.go` and
`apps/worker/main.go`) builds the right `Sender` and returns `nil` if the
selected driver isn't configured — every caller already treats a `nil`
`*email.Mailer` as "email sending disabled", so no other code needs to know
which transport is active.

`SMTPSender` (`pkg/email/smtp.go`) is stdlib-only (`net/smtp`): port `465` is
dialed with implicit TLS, everything else goes through `smtp.SendMail`, which
negotiates STARTTLS automatically when the server advertises it (587 and
most providers) and falls back to plaintext for unauthenticated local dev
catchers (e.g. MailHog/Mailpit on `1025`).

To add a third transport, implement `email.Sender` (one method:
`Send(ctx, Message) error`) and add a case in `NewMailerFromConfig`.

### Sending an email

```go
// Always use LocalizedMailer (i18n-aware), not the raw Mailer:
localMailer.SendVerifyEmail(ctx, lang, to, name, verifyLink, "24")
localMailer.SendPasswordReset(ctx, lang, to, name, resetLink, "60", ip)
localMailer.SendWelcome(ctx, lang, to, name, dashboardURL, docsURL)
localMailer.SendOTPCode(ctx, lang, to, name, code, "10", purpose)
localMailer.SendNotification(ctx, lang, to, name, title, body, ctaLabel, ctaURL)
```

### Adding a new email template

1. Create `email-templates/src/emails/my-template.tsx`
   - Every visible string must be a `__TOKEN__` constant — never hardcode text
   - Data tokens: `__NAME__`, `__LINK__` etc. (caller-supplied values)
   - Text tokens: `__HEADING__`, `__BODY__` etc. (resolved from i18n)

2. Register in `email-templates/src/render.ts` (import + add to `templates` map)

3. Add `locales/en.json` and `locales/id.json` keys for all text tokens

4. Add `TemplateName` constant in `pkg/email/renderer.go`

5. Add `Render*`, `Subject*`, `Send*` methods in `pkg/email/localized_mailer.go`

6. Run `make build-emails`

---

## Multi-language (i18n)

Locale files: `locales/en.json` (default), `locales/id.json`

Key structure:
```
http.*          ← HTTP status messages
auth.*          ← Authentication messages
user.*          ← User CRUD messages
passkey.*       ← WebAuthn messages
mobile.*        ← OTP / mobile verification
appVersion.*    ← App version check messages
notification.*  ← Notification messages
apiKey.*        ← API key messages
tenant.*        ← Multi-tenant messages
validation.*    ← Validator tag messages (with {Field}, {Param} interpolation)
email.*         ← All email template text tokens
  email.layout.*
  email.verifyEmail.*
  email.resetPassword.*
  email.welcome.*
  email.otpCode.*
  email.notification.*
```

**Rules:**
- Add keys to BOTH `en.json` and `id.json`
- Use `{name}`, `{field}` etc. for variable interpolation (NOT Go template syntax)
- Missing keys in non-default languages fall back to `en` automatically

---

## Multi-tenant

Enable with `MULTI_TENANT_ENABLED=true`. Three resolution strategies:

```go
// Header (mobile apps / API clients)
tenantMw.TenantFromHeader(tenantService)  // reads X-Tenant-ID

// Subdomain (web apps)
tenantMw.TenantFromSubdomain(tenantService, cfg.Tenant.BaseDomain)

// Query param (dev/testing)
tenantMw.TenantFromQuery(tenantService)   // reads ?tenant=
```

Tenant context is available as `c.GetString("tenantId")` in handlers.
Pass it to services and repositories when querying tenant-scoped data.

---

## App Versioning

Clients send `X-App-Version: 2.4.1` and `X-App-Platform: ios` headers.
The `VersionCheck` middleware runs on every request:

- `UpdateStatus = "up_to_date"` → pass through, add `X-Version-Status` header
- `UpdateStatus = "available"` → pass through, client can show soft update prompt
- `UpdateStatus = "required"` → **abort with HTTP 426**, client must update

Manage policies at `PUT /v1/admin/app-versions/:platform`.

---

## Passkey / Biometric Authentication

WebAuthn registration and authentication use a two-step ceremony stored in Redis:

```
POST /v1/me/passkeys/register/begin    → returns challenge, stores session in Redis (5 min TTL)
POST /v1/me/passkeys/register/finish   → validates attestation, saves credential to MongoDB
POST /v1/auth/passkey/login/begin      → returns assertion challenge
POST /v1/auth/passkey/login/finish     → validates assertion, updates sign counter, returns tokens
```

`attachment=platform` → biometric (Touch ID / Face ID / Windows Hello)
`attachment=cross-platform` → hardware key or passkey manager

**Never store biometric data** — the server only sees cryptographic assertions.

> **Known gap (TODO):** `modules/passkey/handler/passkey.handler.go`
> `FinishDiscoverableLogin` calls the identified-user `FinishLogin` with a
> `nil` user, so it always returns 401 — discoverable/usernameless passkey
> login is not actually wired up. `BeginIdentifiedLogin`/`FinishIdentifiedLogin`
> are unwired stubs (fixed "Wire UserLoader.FindByEmail…" response, no real
> challenge). Only passkey **registration** (`/v1/me/passkeys/register/*`,
> requires an already-authenticated user) is fully functional today. Fixing
> discoverable login requires threading `webauthn.FinishDiscoverableLogin`'s
> user-resolution callback through instead of reusing the non-discoverable
> `FinishLogin` path. Found while building `web/`, tracked here since it
> isn't visible from the DTOs alone.

---

## Security Middleware Stack (in order)

```
RequestID → Recovery → Logger → SecurityHeaders → Gzip → CORS
  → RateLimiter → VersionCheck → ActivityLogger → i18n
  → [TenantResolver]
  → [APIKeyProtected]     (optional, route-level)
  → [AuthJWTAccess]       (route-level)
  → [RoleProtected]       (route-level)
  → [FeatureFlagProtected] (route-level)
  → Handler
```

Security headers (`middleware/security.go`):
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security` (1 year, includeSubDomains)
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: camera=(), microphone=(), geolocation=()`
- `X-XSS-Protection: 1; mode=block`
- Removes `X-Powered-By`

---

## Swagger Documentation

All handler methods must have Swagger annotations:

```go
// Create godoc
// @Summary      Short description
// @Description  Longer description (optional)
// @Tags         module-name
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path   string              true  "Resource ID"
// @Param        body  body   dto.CreateRequest   true  "Request body"
// @Success      201   {object} dto.Response
// @Failure      400   {object} response.ErrorEnvelope
// @Failure      401   {object} response.ErrorEnvelope
// @Failure      422   {object} response.ErrorEnvelope
// @Router       /v1/resource [post]
```

**Important:** `@BasePath` in `apps/api/main.go` is set to `/`. All `@Router`
paths must include `/v1/` prefix. Never set `@BasePath /v1` — this causes
doubled `/v1/v1/` URLs in Swagger UI.

Run `make swagger` after adding/changing any annotations.

---

## Web Test Client

`web/` is a React + TypeScript SPA (Vite, Tailwind, React Router) that
exercises every HTTP-exposed endpoint of `apps/api` — auth, 2FA, passkeys
(real WebAuthn ceremonies via `@simplewebauthn/browser`), mobile OTP,
notifications, admin users/app-versions/analytics, health — plus a generic
raw-request API console and a client-side activity log for anything not
covered by a dedicated page. It talks to the API directly over `fetch`
(CORS already allows `*`); no backend code changes were needed to support it.

```bash
make web-install     # cd web && npm install
make web-dev          # cd web && vite dev → http://localhost:5173
```

Settings (API base URL, `x-custom-lang`, `X-Tenant-ID`, `X-App-Version`,
`X-App-Platform`) are editable at runtime from the app's own **Settings**
page and persisted in `localStorage` — no rebuild needed to point it at a
different API instance.

**WebAuthn origin:** `WEBAUTHN_RP_ORIGIN` must equal the exact origin that
calls `navigator.credentials.*` in the browser — that's `web/` at
`http://localhost:5173`, not the API's own `:3000`. `.env.example` already
defaults to `5173`; if you serve `web/` from a different host/port, update
`WEBAUTHN_RP_ORIGIN` and restart the API or passkey ceremonies will fail.

**GitHub OAuth in the test client:** `GET /v1/auth/github/callback` returns
the login JSON directly instead of redirecting into the SPA, so the login
button opens it in a new tab — the OAuth page has a "paste JSON" box to
import that session back into the app.

See `web/README.md` for the full breakdown, and the passkey known-gap note
in the Passkey section above (surfaced while wiring this client's WebAuthn
pages).

---

## Testing

### Unit tests
Tests live next to their implementation (same package, `_test.go` suffix):
```
modules/user/service/user.service_test.go
shared/pagination/pagination_test.go
modules/appversion/service/appversion_test.go
```

Use fake/stub implementations, never real MongoDB/Redis in unit tests.

### Integration tests
`tests/integration/` — use `httptest.NewRecorder()` and `gin.CreateTestContext()`.
Skip tests that need JWT keys with `t.Skip("Requires JWT key files")`.

### E2E tests
`tests/e2e/` — hits live server. Auto-skips if server not running:
```bash
make docker-up && make seed
go test ./tests/e2e/... -v
```

### Run tests
```bash
make test              # all unit tests (no DB needed)
make test-unit         # short mode
make test-e2e          # requires running stack
make test-coverage     # HTML report
```

---

## Environment Variables

Key variables (see `.env.example` for complete list):

```env
# Required
APP_ENV=development
MONGO_URI=mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0
REDIS_HOST=redis
JWT_ACCESS_PRIVATE_KEY_PATH=./keys/access_private.pem
JWT_ACCESS_PUBLIC_KEY_PATH=./keys/access_public.pem
JWT_REFRESH_PRIVATE_KEY_PATH=./keys/refresh_private.pem
JWT_REFRESH_PUBLIC_KEY_PATH=./keys/refresh_public.pem

# Optional integrations
MAIL_DRIVER=ses                # "ses" (default, needs AWS_ACCESS_KEY_ID/AWS_SES_*) or "smtp" (needs SMTP_*)
AWS_ACCESS_KEY_ID=             # enables SES email (if MAIL_DRIVER=ses) + S3
TWILIO_ACCOUNT_SID=            # enables SMS + WhatsApp OTP
FIREBASE_CREDENTIALS_FILE=     # enables FCM push notifications
GITHUB_CLIENT_ID=              # enables GitHub SSO
APPLE_CLIENT_ID=               # enables Apple Sign In
SENTRY_DSN=                    # enables error tracking
MULTI_TENANT_ENABLED=false     # enable multi-tenant mode
```

In Docker, `MONGO_URI` and `REDIS_HOST` are always overridden by
`docker-compose.yml` `environment:` block to use Docker service names.

**Custom env file for local scripts:**
```bash
# make seed-local and make migrate-indexes-local read .env by default.
# Override with:
make seed-local ENV_FILE=.env.staging
# Or directly:
BREAD_CONFIG_FILE=/path/to/.env go run ./scripts/seed/main.go
```

---


## Dev Container (VS Code / GitHub Codespaces)

The project ships a fully configured dev container. Open in VS Code with the
**Dev Containers** extension installed — everything is set up automatically.

### What the dev container provides

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.23 | Compiler, go tools |
| Node.js | 20 | React Email builds + `web/` test client |
| air | latest | Hot reload (`make dev`) |
| golangci-lint | latest | Linting (`make lint`) |
| goimports | latest | Import formatting |
| mongosh | 8.0 | MongoDB shell |
| redis-cli | bundled | Redis inspection |
| Docker CLI | latest | `make seed`, `make docker-*` |

### Services started automatically

| Service | Address | Purpose |
|---|---|---|
| MongoDB rs0 | `mongo1:27017` | Primary database |
| Redis | `redis:6379` | Cache + sessions |
| (API) | `localhost:3000` | Started manually via `make dev` |
| (Web test client) | `localhost:5173` | Started manually via `make web-dev` |

### First-time open

1. Install the **Dev Containers** extension in VS Code
2. Open the project folder: VS Code detects `.devcontainer/` and prompts  
   → Click **"Reopen in Container"**
3. Container builds + `post-create.sh` runs automatically:
   - Generates JWT key pairs
   - Patches `.env` with correct service URIs
   - Builds React Email templates
   - Generates Swagger docs
   - Runs `go mod tidy`
   - Installs `web/` test client dependencies
4. Start the API: `make dev`
5. Seed the DB: `make seed`
6. Browse API docs: http://localhost:3000/docs
7. Start the test client: `make web-dev` → http://localhost:5173

### VS Code Tasks (Ctrl+Shift+B or Terminal → Run Task)

- 🚀 **Start dev server** — `make dev` with hot reload
- 🧪 **Run all tests** — `make test`
- 📖 **Generate Swagger** — `make swagger`
- 📧 **Build email templates** — `make build-emails`
- 🌱 **Seed database** — `make seed`
- 🖥️ **Start web test client** — `make web-dev`

### Debugger

Press **F5** to launch the API with the Go debugger attached.
Set breakpoints in any handler, service, or repository file.

### GitHub Codespaces

The same `.devcontainer/` config works with Codespaces — push to GitHub
and click **"Open in Codespaces"**. All services start automatically.

---

## Docker

```bash
make docker-up          # start: API + MongoDB 3-node rs + Redis
make docker-worker      # same + Asynq background worker
make docker-down        # stop all
make docker-rebuild     # rebuild + restart app container only
make docker-clean       # remove containers AND volumes (destructive)
make seed               # seed DB (runs inside Docker network, requires docker-up)
make migrate-indexes    # create MongoDB indexes (inside Docker network)
make seed-local         # seed against localhost:27017 (for local dev without Docker)
```

The `email-builder` Docker stage compiles React Email templates before the
Go build stage runs, so the embedded HTML is always fresh in Docker builds.

---

## MongoDB Requirements

- **Replica set required** — transactions need `rs0` with 3 nodes
- The app retries ping up to 10 times × 3s to handle replica set election delay
- Indexes are declared in `scripts/migrate/main.go` — run after first deploy
- TTL indexes auto-expire: activity logs (90 days), notifications (90 days)
- Soft deletes: users have `deletedAt` field; always filter `deletedAt: nil`

---

## Common Pitfalls

| Pitfall | Correct approach |
|---|---|
| Using `c.ShouldBindJSON` directly in a handler | Use `validate.BindJSON(c, &req)` |
| Returning raw Go error strings to the client | Wrap in `errors.AppError` or use `handleError(c, err)` |
| Hardcoding English text in handlers | Use `pkgi18n.TC(c, "key")` |
| Hardcoding text in email templates | Every visible string must be a `__TOKEN__` |
| `@BasePath /v1` in swagger annotations | Use `@BasePath /` — routes already include `/v1` |
| Running `make seed` without `make docker-up` | Seed runs inside Docker network — start stack first |
| `go:embed` with `../` paths | Embedded files must be inside or below the package directory |
| Importing the same package with different aliases in the same package | Use consistent alias across all files in the package |
| Using `primitive.NewObjectID()` for new documents | Use `uuid.New()` — all `_id` fields are `uuid.UUID` |
| Using `primitive.ObjectIDFromHex(id)` to parse URL params | Use `uuid.Parse(id)` — returns `(uuid.UUID, error)` |
| Importing `go.mongodb.org/mongo-driver/bson/primitive` in entities/repos | Only allowed in `pkg/uuidbson/codec.go` |
| Adding a field to `Config` struct without adding it to `Load()` | Always add both struct field AND `v.GetString()/GetBool()` call in `Load()` |
| JWT `@BasePath /v1` causing `/v1/v1/auth/login` | Always `@BasePath /` |
| `go mod tidy` hanging on `docs/swagger` package | `make swagger` MUST run before `make mod-sync` — `make setup` does this automatically |
| `GET /v1/me` returns 400 instead of 401 | Auth middleware not applied — check every `RegisterRoutes` passes `authMw` to protected groups |
| `GET /v1/users` returns 403 with valid admin token | JWT `role` claim is a UUID not a slug — `AuthService` needs `RoleRepo` to resolve slug at login time |
| `$set` and `$setOnInsert` conflict in seed script | Same field in both operators — put `_id`/`createdAt`/unique keys in `$setOnInsert`, everything else in `$set` |
| `make seed-local` ignoring `.env` settings | Pass `BREAD_CONFIG_FILE=$(ENV_FILE)` to the script; Viper reads the file path from that env var |
| `docs/swagger/docs.go` missing after fresh clone | Only `docs/swagger/*.json` and `*.yaml` are gitignored — `docs.go` stub is always committed |
| Importing `modules/X/service` from `modules/Y/` | Cross-module imports must go through `modules/X/contract.go` only |
| Forgetting to add `contract.go` to a new module | Define the public interface first — it enforces the service boundary |
| `DB_ID_TYPE` mismatch between .env and entity field types | UUID and ObjectID require different Go types in entities — see `docs/id-migration.md` |
| Passkey ceremonies fail with a WebAuthn `SecurityError` in the browser | `WEBAUTHN_RP_ORIGIN` must match the origin calling `navigator.credentials.*` (the `web/` test client, `:5173`) — not the API's own `:3000` |
| Testing `web/` against a non-default API port/host | Change the API base URL from the app's own Settings page (persisted in `localStorage`) — no rebuild needed |
| Setting `SMTP_*` vars but email still not sending | `MAIL_DRIVER` must be set to `smtp` — it stays on `ses` (the default) otherwise, and SMTP config is ignored |
| Constructing a `*email.Mailer` directly with `email.NewMailer(cfg.AWS)` | `NewMailer` now takes an `email.Sender`, not `AWSConfig` — use `email.NewMailerFromConfig(cfg, log)` |

---

## Key Files to Know

| File | Purpose |
|---|---|
| `apps/api/app/app.go` | All dependency wiring — edit when adding a module |
| `shared/config/config.go` | Add new env vars (struct + Load() + SetDefault + viper default) |
| `shared/validate/validate.go` | Single validation entry point for all handlers |
| `shared/response/response.go` | Standard response envelope helpers |
| `shared/errors/errors.go` | Domain error sentinels (add new ones here) |
| `modules/role/repository/role.repository.go` | Role lookup — `FindByID` used during JWT issuance |
| `modules/activity/service/activity.service.go` | Action constants + log helpers |
| `modules/<name>/contract.go` | Public interface — the only safe cross-module import |
| `pkg/dbid/dbid.go` | ID strategy helpers (uuid / objectid) — see `docs/id-migration.md` |
| `pkg/email/localized_mailer.go` | Render*/Send* email methods |
| `pkg/email/factory.go` | `NewMailerFromConfig` — picks SES vs SMTP via `MAIL_DRIVER` |
| `locales/en.json` + `locales/id.json` | All i18n strings including email tokens |
| `scripts/migrate/main.go` | All MongoDB index definitions |
| `scripts/seed/main.go` | Default data (roles, users, flags, app versions) |
| `docs/id-migration.md` | How to switch between UUID and ObjectID |
| `docs/email-i18n.md` | Email template architecture documentation |
| `docs/feature-parity.md` | Full NestJS ↔ Go feature mapping |
| `web/README.md` | Web test client — running it, WebAuthn origin setup, OAuth caveats |
| `web/src/lib/apiClient.ts` | Test client's fetch wrapper — auth headers, refresh-on-401, request logging |
| `web/src/api/*.ts` | Typed wrappers per module, mirroring each `dto` package 1:1 |
