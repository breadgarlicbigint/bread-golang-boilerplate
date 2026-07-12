# Bread Golang Boilerplate 🔥 🚀

A production-ready **Go** REST API boilerplate inspired by [ack-nestjs-boilerplate](https://github.com/andrechristikan/ack-nestjs-boilerplate), rebuilt with idiomatic Go patterns.

---

## Tech Stack

| Concern | NestJS original | Go equivalent |
|---|---|---|
| HTTP framework | NestJS / Express | **Gin** |
| ORM / Driver | Prisma | **mongo-driver** (Repository Pattern) |
| Database | MongoDB (replica set) | **MongoDB 8** (replica set) |
| Cache / Sessions | Redis | **go-redis** |
| Auth tokens | JWT ES256 / ES512 | **golang-jwt** ES256 / ES512 |
| Password hashing | bcrypt | **bcrypt** (golang.org/x/crypto) |
| 2FA | TOTP (speakeasy) | **pquerna/otp** |
| Passkey / WebAuthn | — | **go-webauthn/webauthn** (FIDO2) |
| Background jobs | BullMQ | **asynq** (Redis, default) — or **RabbitMQ** / **Kafka** via `QUEUE_DRIVER` |
| Config | @nestjs/config | **viper** |
| Validation | class-validator | **go-playground/validator** |
| Logging | Winston | **zap** (Uber) |
| API docs | Swagger (NestJS) | **swaggo/swag** + **gin-swagger** |
| Error tracking | Sentry | **sentry-go** |
| Email delivery | AWS SES | **aws-sdk-go-v2 SES** or **stdlib SMTP** — selected via `MAIL_DRIVER` |
| Email templates | — | **React Email** (build-time, compiled to embedded HTML) |
| SMS / WhatsApp OTP | — | **twilio/twilio-go** |
| Push notifications | — | **Firebase FCM** (`firebase.google.com/go/v4`) |
| File storage | AWS S3 | **aws-sdk-go-v2 S3** |
| OAuth / SSO | passport-google | **golang.org/x/oauth2** (GitHub) + Sign in with Apple |
| i18n | — | Custom **`pkg/i18n`** (`locales/*.json`, `x-custom-lang` header) |
| Multi-tenancy | — | Header / subdomain / query-param tenant resolution |
| Hot reload | ts-node-dev | **air** (API + all three worker processes) |
| Containerisation | Docker | **Docker** multi-stage + dev container (VS Code/Codespaces) |

---

## Features

### 🎯 Architecture
- **Repository Pattern** — clean data access abstraction (no ORM lock-in)
- **Modular monorepo** — `modules/<module>/{contract.go,entity,dto,repository,service,handler}`, each module importable only through its `contract.go` interface (microservice-extraction-ready)
- **SOLID principles** — interfaces everywhere, easy to mock & test
- **12-Factor App** compliant config via environment variables

### 🔐 Authentication & Security
- JWT **ES256** access tokens + **ES512** refresh tokens (asymmetric keys)
- **Stateful sessions** — Redis-backed with per-session revocation
- **Token rotation** — new pair issued on every refresh
- **TOTP 2FA** with backup codes
- **Passkey / Biometric login** (WebAuthn/FIDO2) — Touch ID, Face ID, Windows Hello, hardware keys (registration fully wired; see the passkey login caveat below)
- **GitHub SSO** and **Sign in with Apple**
- **RBAC** — role + permission checks via middleware, roles resolved from Mongo (`GET /v1/roles` populates admin UI dropdowns)
- **API Key** auth (hashed with bcrypt, prefix-indexed lookup)
- **Rate limiting** — sliding-window counter per IP via Redis
- **Brute-force protection** — automatic account lockout
- **Multi-tenant** — header / subdomain / query-param tenant resolution, opt-in via `MULTI_TENANT_ENABLED`
- CORS, Gzip, security headers (HSTS, X-Frame-Options, CSP-adjacent headers, `X-Powered-By` removed)

### 📊 Database & Storage
- MongoDB with **transaction support** (requires replica set) and **UUID primary keys** (BSON Binary subtype 4, not ObjectID)
- **TTL indexes** on activity logs and notifications (auto-expiry after 90 days)
- Redis **multi-level caching** (feature flags, sessions, analytics)
- **AWS S3** — upload, presigned upload/download, delete
- **Soft deletes** on users

### ⚡ Performance
- Background job queue (email, notifications, cleanup tasks) — **Redis/Asynq** by default, swappable for **RabbitMQ** or **Kafka** via `QUEUE_DRIVER` (see "Background Job Queue Backends" below)
- Response **gzip compression**
- **Cursor + offset** pagination with metadata
- Per-collection MongoDB indexes

### 🛠 Developer Experience
- **Hot reload** via `air` — API (`make dev`) and all three worker processes (`make dev-worker[-rabbitmq|-kafka]`)
- **Dev Container** (VS Code / GitHub Codespaces) — Go, Node, air, golangci-lint, mongosh, Docker CLI, MongoDB rs + Redis auto-started
- **Web test client** (`web/`) — React/TS SPA exercising every endpoint, incl. real WebAuthn ceremonies and an i18n side-by-side comparison page
- `make setup` one-command bootstrap
- `make generate-keys` — EC key pair generation
- `make swagger` — Swagger docs generation
- `make seed` — database seeding (roles, users, flags, app versions)
- `make test-coverage` — HTML coverage report
- golangci-lint config

### 📡 Integrations
- **Sentry** error tracking + traces
- **Transactional email** — AWS SES or SMTP (selected via `MAIL_DRIVER`), React Email templates, bilingual (`en`/`id`) via `pkg/i18n`, delivered asynchronously through the job queue
- **SMS & WhatsApp OTP** — Twilio, for mobile number verification
- **Push notifications** — Firebase Cloud Messaging, plus in-app notification feed with read/unread state and per-user preferences
- **Activity logging** — bidirectional (inbound + outbound) with 90-day auto-expiry
- **Feature flags** (MongoDB + Redis cache)
- **Health checks** — `/health`, `/health/live`, `/health/ready`

---

## Project Structure

```
bread-golang-boilerplate/
├── apps/                        # Deployable binaries
│   ├── api/                     # HTTP server entry point + Gin/module wiring (app/app.go)
│   ├── worker/                  # Asynq (Redis) background worker
│   ├── worker-rabbitmq/         # RabbitMQ background worker (same task handlers)
│   └── worker-kafka/            # Kafka background worker (same task handlers)
├── modules/                     # Domain modules (future microservices)
│   ├── auth/                    # Login, Register, Refresh, Logout, 2FA, GitHub/Apple SSO
│   ├── user/                    # CRUD, block/unblock, password change
│   ├── role/                    # RBAC roles + permissions
│   ├── apikey/                  # API key hashing/lookup (service-only, no REST yet)
│   ├── activity/                # Bidirectional audit log
│   ├── analytics/               # Admin analytics + fraud/anomaly scoring
│   ├── appversion/               # Forced-update policy per platform
│   ├── featureflag/              # Feature flags with Redis cache (service-only, no REST yet)
│   ├── health/                   # Liveness + readiness probes
│   ├── mobile/                   # SMS/WhatsApp OTP verification
│   ├── notification/             # In-app notifications + FCM push + admin send/broadcast
│   ├── passkey/                  # WebAuthn/FIDO2 registration + login
│   └── tenant/                   # Multi-tenant resolution (header/subdomain/query)
│       # each module: contract.go entity/ dto/ repository/ service/ handler/
├── shared/                      # Cross-cutting infrastructure (no business logic)
│   ├── config/                  # Viper config loader
│   ├── database/                # MongoDB + Redis clients
│   ├── errors/                  # Domain error types + sentinels
│   ├── middleware/               # Auth, API Key, Rate limit, Security, Feature flag, Activity
│   ├── pagination/               # Offset + cursor helpers
│   ├── response/                 # Standard JSON envelope + i18n-aware helpers
│   └── validate/                 # Centralised validate.BindJSON/BindQuery
├── pkg/                          # Pure utilities (zero domain knowledge)
│   ├── dbid/                     # ID strategy helpers (uuid / objectid)
│   ├── email/                    # SES/SMTP mailer + React Email renderer + LocalizedMailer
│   ├── hash/                     # bcrypt wrapper
│   ├── i18n/                     # Locale loader + Translator + Gin middleware
│   ├── jwt/                      # ES256/ES512 token manager
│   ├── logger/                   # Zap factory
│   ├── sms/                      # Twilio SMS + WhatsApp sender
│   ├── storage/                  # AWS S3 client
│   ├── uuidbson/                 # UUID BSON Binary-4 codec
│   ├── worker/                   # Asynq (Redis) client + server + task handlers
│   └── queue/                    # Backend-agnostic Publisher/Consumer contract
│       ├── router/               # Routes task types to different Publishers (transactional/promotional)
│       ├── rabbitmq/             # RabbitMQ (AMQP) implementation
│       ├── kafka/                # Kafka implementation
│       └── tasks/                # Shared handlers for the RabbitMQ/Kafka workers
├── locales/                      # en.json, id.json — i18n strings incl. email tokens
├── email-templates/              # React Email .tsx source (build-time, Node.js only)
├── web/                          # React + TS test client (Vite) — see "Web Test Client" below
├── scripts/
│   ├── seed/                     # Seed roles, users, feature flags, app versions
│   └── migrate/                  # Create MongoDB indexes
├── docs/                         # Architecture docs, id-migration, email-i18n, swagger
├── keys/                         # EC key pairs (git-ignored)
├── .devcontainer/                # VS Code / Codespaces dev container
├── .env.example
├── .air.toml                     # API hot reload
├── .air.worker.toml              # Redis/Asynq worker hot reload
├── .air.worker-rabbitmq.toml     # RabbitMQ worker hot reload
├── .air.worker-kafka.toml        # Kafka worker hot reload
├── .golangci.yml
├── docker-compose.yml
├── Dockerfile
├── Dockerfile.worker
└── Makefile
```

---

## Quick Start

### Prerequisites
- Go 1.23+
- Docker & Docker Compose
- OpenSSL (for key generation)

```bash
# 1. Clone
git clone https://github.com/breadgarlicbigint/bread-golang-boilerplate
cd bread-golang-boilerplate

# 2. One-command setup (generates keys, copies .env, tidies modules)
make setup

# 3. Edit environment variables
nano .env

# 4. Start MongoDB replica set + Redis + app
make docker-up

# 5. Seed the database
make seed

# 6. API is live
open http://localhost:3000/health
open http://localhost:3000/docs/index.html
```

### Development (hot reload)
```bash
# Install air
go install github.com/air-verse/air@latest

# Start API with hot reload
make dev

# Start a worker with hot reload (pick the one matching QUEUE_DRIVER)
make dev-worker            # Redis/Asynq worker
make dev-worker-rabbitmq   # RabbitMQ worker
make dev-worker-kafka      # Kafka worker
```

`make dev`'s build step (`scripts/dev/air-build-api.sh`) regenerates Swagger
docs on every reload and rebuilds compiled email templates whenever a file
under `email-templates/src` changed — no separate `make swagger` /
`make build-emails` needed while iterating on handlers or `.tsx` templates.

---

## Key Management

Access and refresh tokens use **EC asymmetric keys** — the private key signs, the public key verifies. This means microservices can verify tokens without ever holding the private key.

```bash
make generate-keys
# Writes to ./keys/ (git-ignored)
# access_private.pem  — ES256 (P-256)
# access_public.pem
# refresh_private.pem — ES512 (P-521)
# refresh_public.pem
```

---

## API Overview

### Auth (`/v1/auth`)
| Method | Path | Auth |
|--------|------|------|
| POST | `/login` | Public |
| POST | `/register` | Public |
| POST | `/refresh` | Public |
| DELETE | `/logout` | Bearer |
| DELETE | `/logout-all` | Bearer |
| POST | `/2fa/enable` | Bearer |
| POST | `/2fa/verify` | Bearer |
| GET | `/github` | Public — redirects to GitHub |
| GET | `/github/callback` | Public — GitHub redirect target |
| POST | `/apple/callback` | Public — Sign in with Apple |
| POST | `/passkey/login/begin` · `/finish` | Public — usernameless (discoverable) login |
| POST | `/passkey/identified/begin` · `/finish` | Public — user-identified login |

### Passkeys (`/v1/me/passkeys`) — Bearer
| Method | Path |
|--------|------|
| POST | `/register/begin` |
| POST | `/register/finish` |
| GET | `` |
| DELETE | `/:id` |

### Users (`/v1`)
| Method | Path | Auth |
|--------|------|------|
| POST | `/users` | Admin |
| GET | `/users` | Admin |
| GET | `/users/:id` | Admin |
| PATCH | `/users/:id` | Admin |
| DELETE | `/users/:id` | Admin |
| POST | `/users/:id/block` | Admin |
| POST | `/users/:id/unblock` | Admin |
| GET | `/me` | Bearer |
| PATCH | `/me` | Bearer |
| PATCH | `/me/password` | Bearer |

### Roles (`/v1/roles`)
| Method | Path | Auth |
|--------|------|------|
| GET | `` | Admin — populates role-selection dropdowns |

### Mobile verification (`/v1/me/mobiles`) — Bearer
| Method | Path |
|--------|------|
| GET | `` |
| POST | `/send-otp` |
| POST | `/verify` |
| PATCH | `/:e164/primary` |
| DELETE | `/:e164` |

### Notifications (`/v1`)
| Method | Path | Auth |
|--------|------|------|
| GET | `/me/notifications` | Bearer |
| GET | `/me/notifications/unread-count` | Bearer |
| PATCH | `/me/notifications/:id/read` | Bearer |
| PATCH | `/me/notifications/read-all` | Bearer |
| GET | `/me/notifications/preferences` | Bearer |
| PATCH | `/me/notifications/preferences` | Bearer |
| POST | `/me/notifications/devices` | Bearer — register FCM device token |
| DELETE | `/me/notifications/devices/:token` | Bearer |
| POST | `/admin/notifications/send` | Admin |
| POST | `/admin/notifications/broadcast` | Admin |
| POST | `/admin/notifications/test-email` | Admin — synchronous, returns raw transport error for diagnostics |

### App Versioning (`/v1`)
| Method | Path | Auth |
|--------|------|------|
| GET | `/app-version/check` | Public — mobile clients call on launch |
| GET | `/admin/app-versions` | Admin |
| PUT | `/admin/app-versions/:platform` | Admin |

### Analytics (`/v1/admin/analytics`) — Admin
| Method | Path |
|--------|------|
| GET | `/users/registrations` · `/churn` · `/signup-methods` · `/blocked-trend` |
| GET | `/auth/login-frequency` · `/login-methods` · `/lockout` |
| GET | `/passkeys/adoption` |
| GET | `/mobile/verification` |
| GET | `/anomalies/credential-stuffing` · `/device-proliferation` |
| GET | `/fraud/signals` |

### Health
| Method | Path |
|--------|------|
| GET | `/health` |
| GET | `/health/live` |
| GET | `/health/ready` |

> **Not yet exposed via REST:** `apikey` and `featureflag` are fully
> implemented at the service layer (used internally by
> `APIKeyProtected`/`FeatureFlagProtected` middleware) but have no
> `handler/` package or admin CRUD routes yet — manage them via
> `scripts/seed` / direct DB writes for now.
>
> **Known gap:** discoverable (`/passkey/login/*`) and identified
> (`/passkey/identified/*`) passkey login are wired as routes but not
> functionally complete server-side yet — only passkey **registration**
> (`/me/passkeys/register/*`) is fully working today. See
> `modules/passkey/handler/passkey.handler.go` and the passkey note in
> `CLAUDE.md`.

---

## Response Envelope

All endpoints return a consistent JSON envelope:

```json
{
  "statusCode": 200,
  "message": "User fetched",
  "data": { ... },
  "_metadata": {
    "total": 100,
    "page": 1,
    "perPage": 20,
    "totalPage": 5,
    "hasNext": true,
    "hasPrev": false
  },
  "timestamp": "2026-01-01T00:00:00Z",
  "path": "/v1/users",
  "requestId": "uuid-here"
}
```

Every handler response goes through the i18n-aware helpers
(`response.OKI18n`, `response.CreatedI18n`, `response.ErrorI18n`, ...) which
take a `locales/*.json` key and translate `message` based on the
`x-custom-lang` request header — no hardcoded client-facing strings in
handlers. Errors go through `response.HandleAppError`, which unwraps
`*errors.AppError` (status + translated message) and falls back to a
generic translated 500 for anything else.

---

## Middleware Stack (in order)

```
RequestID → Recovery → Logger → SecurityHeaders → Gzip → CORS
  → RateLimiter → VersionCheck → ActivityLogger → i18n
  → [TenantResolver]
  → [APIKeyProtected]      (optional, route-level)
  → [AuthJWTAccess]        (route-level)
  → [RoleProtected]        (route-level)
  → [FeatureFlagProtected] (route-level)
  → Handler
```

---

## Testing

```bash
make test              # all unit tests (no DB needed)
make test-unit         # short mode
make test-integration  # tests/integration — httptest + gin.CreateTestContext (skips if JWT keys missing)
make test-e2e          # tests/e2e — hits a live server, requires: make docker-up && make seed
make test-coverage     # HTML coverage report → coverage.html
```

---

## Environment Variables

See [`.env.example`](.env.example) for the full list with descriptions. Key
optional integrations — each is disabled gracefully if unconfigured:

| Variable | Enables |
|---|---|
| `MAIL_DRIVER` (`ses` \| `smtp`) | Transactional email — AWS SES (`AWS_ACCESS_KEY_ID`/`AWS_SES_*`) or SMTP (`SMTP_*`) |
| `AWS_ACCESS_KEY_ID` | SES email (if `MAIL_DRIVER=ses`) + S3 storage |
| `TWILIO_ACCOUNT_SID` | SMS + WhatsApp OTP delivery |
| `FIREBASE_CREDENTIALS_FILE` | FCM push notifications |
| `GITHUB_CLIENT_ID` | GitHub SSO |
| `APPLE_CLIENT_ID` | Sign in with Apple |
| `SENTRY_DSN` | Error tracking |
| `MULTI_TENANT_ENABLED` | Multi-tenant mode (header/subdomain/query resolution) |
| `QUEUE_DRIVER` (`redis` \| `rabbitmq` \| `kafka`) | Default background job backend — must match a worker process you run |
| `QUEUE_TRANSACTIONAL_DRIVER` / `QUEUE_PROMOTIONAL_DRIVER` | Optional per-workload override of `QUEUE_DRIVER` — see "Routing different workloads to different brokers" below |

---

## License

MIT

---

## Added Features (v2)

### 1. Bidirectional Activity Logging

Every HTTP request generates **two** linked log entries sharing a `correlationId`:

| Direction | What it records |
|---|---|
| `inbound` | What the client/user DID (method, path, actor, IP, UA) |
| `outbound` | What the system DID BACK (status, latency, email sent, SMS delivered) |

The `ActivityLogger` middleware auto-wires both directions on every request. Individual services call `LogEmailSent`, `LogSMSSent`, `LogAPIResponse`, etc. for richer outbound detail.

```
GET /v1/users/123  →  inbound log: { action: "apiResponse", actorId, ip, ... }
                   →  outbound log: { action: "apiResponse", status: 200, latencyMs: 12, ... }
```

New MongoDB indexes: `correlationId`, `direction`, `channel`, `tenantId`, `actorId+direction`.

---

### 2 & 3. Passkey + Biometric Login (WebAuthn/FIDO2)

Full **WebAuthn Level 2** implementation using `go-webauthn/webauthn`.

**How biometrics work server-side:**
> Touch ID, Face ID, and Windows Hello are *platform authenticators* — they generate a FIDO2 passkey bound to the device. The server never sees the biometric data; it only verifies the cryptographic assertion. Pass `attachment=platform` to get a biometric credential; `cross-platform` for hardware keys or password managers.

| Flow | Endpoints |
|---|---|
| Register passkey | `POST /v1/me/passkeys/register/begin` → `POST /v1/me/passkeys/register/finish` |
| List passkeys | `GET /v1/me/passkeys` |
| Delete passkey | `DELETE /v1/me/passkeys/:id` |
| Usernameless login | `POST /v1/auth/passkey/login/begin` → `POST /v1/auth/passkey/login/finish` |
| User-identified login | `POST /v1/auth/passkey/identified/begin` → `/finish` |

WebAuthn ceremony state is stored in Redis with a 5-minute TTL. Sign counters are updated after every successful assertion to detect cloned authenticators.

> ⚠️ **Known gap:** only passkey **registration** is fully functional today.
> `FinishDiscoverableLogin` currently calls the identified-user `FinishLogin`
> path with a `nil` user, so it always returns 401 — discoverable
> (usernameless) login isn't actually wired up, and
> `BeginIdentifiedLogin`/`FinishIdentifiedLogin` are unwired stubs. Fixing
> this requires threading `webauthn.FinishDiscoverableLogin`'s
> user-resolution callback through instead of reusing the non-discoverable
> `FinishLogin` path. See `modules/passkey/handler/passkey.handler.go`.

---

### 4. GitHub SSO & Sign in with Apple

**GitHub** — full OAuth2 flow with CSRF state protection:

```
GET  /v1/auth/github           →  redirects to GitHub
GET  /v1/auth/github/callback  →  exchanges code, auto-registers or links account, returns token pair
```

- State token stored in Redis (10-min TTL)  
- Fetches primary verified email from `/user/emails` if profile email is private  
- Auto-links to existing account if email matches  

**Apple** — Sign in with Apple via the identity token flow:

```
POST /v1/auth/apple/callback   →  verifies Apple's signed identity token, auto-registers or links account, returns token pair
```

Configure via `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`,
`APPLE_PRIVATE_KEY_PATH` — the handler is only wired if `APPLE_CLIENT_ID` is set.

---

### 5. Analytics Dashboard

All endpoints cached in Redis (1-hour TTL, busted on demand). `X-Cache: HIT/MISS` header always returned.

| Category | Endpoints |
|---|---|
| Users | `GET /v1/admin/analytics/users/registrations` · `/churn` · `/signup-methods` · `/blocked-trend` |
| Auth | `GET /v1/admin/analytics/auth/login-frequency` · `/login-methods` · `/lockout` |
| Passkeys | `GET /v1/admin/analytics/passkeys/adoption` |
| Mobile | `GET /v1/admin/analytics/mobile/verification` |
| Anomalies | `GET /v1/admin/analytics/anomalies/credential-stuffing` · `/device-proliferation` |
| Fraud | `GET /v1/admin/analytics/fraud/signals` |

**Fraud scoring** follows the spec's weighted signal model:

| Signal | Weight |
|---|---|
| Approaching lockout | +20 |
| Impossible travel | +30 |
| New device + password change | +20 |
| Multiple failed 2FA | +15 |

Risk levels: `monitor` (15–30) → `review` (31–60) → `reauth` (61–90) → `suspend` (91+)

---

### 6. Multi-Tenant Architecture

**Tenant resolution strategies** (choose one or combine):

| Strategy | Middleware | Header/Source |
|---|---|---|
| Header | `TenantFromHeader` | `X-Tenant-ID: acme-corp` |
| Subdomain | `TenantFromSubdomain` | `acme-corp.yourapp.com` |
| Custom domain | Auto via subdomain MW | `app.acme.com` |
| Query param | `TenantFromQuery` | `?tenant=acme-corp` |

Enable with `MULTI_TENANT_ENABLED=true`. All user data, activity logs, and passkeys include `tenantId`. Plans: `free` (5 users, 14-day trial) → `starter` (5) → `pro` (50) → `enterprise` (1000).

---

### 6b. Email System — React Email + Multi-language

All transactional emails use **React Email** components compiled to embedded HTML,
with every visible string resolved from the Go i18n system at send time.

**Transport:** `MAIL_DRIVER` selects **AWS SES** (default, `AWS_ACCESS_KEY_ID`/`AWS_SES_*`)
or **SMTP** (`SMTP_HOST`/`SMTP_PORT`/`SMTP_USERNAME`/`SMTP_PASSWORD`/`SMTP_FROM_*`) —
stdlib `net/smtp` only, implicit TLS on `465`, STARTTLS auto-negotiated on `587`,
plaintext fallback for local dev catchers (MailHog/Mailpit on `1025`).
`NewMailerFromConfig` returns `nil` if the selected driver isn't configured, and
every caller treats a `nil` mailer as "email sending disabled".

**Delivery:** transactional emails are rendered synchronously in the API
(fast, needs the request's `x-custom-lang`) then enqueued via the
`EmailQueue` interface (`EnqueueEmail`) and actually sent by whichever
worker process matches `QUEUE_DRIVER` — see "Background Job Queue Backends"
below. The one exception is `POST /v1/admin/notifications/test-email`, which
sends synchronously so mail-driver config can be verified without a worker
running.

**5 templates:** `verify-email` · `reset-password` · `welcome` · `otp-code` · `notification`

**2 languages built-in:** English (`en`) · Indonesian (`id`) · _add more by creating `locales/xx.json`_

```go
// Send in the user's preferred language (from x-custom-lang header):
_, lang := pkgi18n.FromContext(c)
authSvc.SendWelcomeEmail(ctx, lang, email, name, appURL)
authSvc.SendVerificationEmail(ctx, lang, userID, email, name, appURL)
authSvc.SendPasswordResetEmail(ctx, lang, userID, email, name, appURL, ip)

// Or use LocalizedMailer directly:
localMailer.SendOTPCode(ctx, "id", to, name, "483920", "10", "autentikasi dua faktor")
```

Build templates after editing `.tsx` files:
```bash
make build-emails   # requires Node.js
```

See [docs/email-i18n.md](docs/email-i18n.md) for full documentation.

---

### 7. Mobile Number Verification

OTP delivery via **SMS** or **WhatsApp** (Twilio).

```
POST /v1/me/mobiles/send-otp  { "e164": "+15551234567", "channel": "whatsapp" }
POST /v1/me/mobiles/verify    { "e164": "+15551234567", "code": "483920" }
GET  /v1/me/mobiles
PATCH /v1/me/mobiles/:e164/primary
DELETE /v1/me/mobiles/:e164
```

- 6-digit numeric OTP, 10-minute TTL in Redis  
- Max 5 attempts per OTP before auto-invalidation  
- OTP stored as bcrypt hash (never plaintext in Redis)  
- Every send/fail logged as outbound activity  

---

### 7b. Push Notifications & In-App Notification Feed

In-app notification feed backed by MongoDB (TTL: 90 days), with optional
**Firebase Cloud Messaging** push delivery to registered devices.

```
GET    /v1/me/notifications                → list (paginated, ?unreadOnly=true)
GET    /v1/me/notifications/unread-count
PATCH  /v1/me/notifications/:id/read
PATCH  /v1/me/notifications/read-all
GET    /v1/me/notifications/preferences
PATCH  /v1/me/notifications/preferences
POST   /v1/me/notifications/devices         → register an FCM device token
DELETE /v1/me/notifications/devices/:token

POST   /v1/admin/notifications/send         → send to one user (admin)
POST   /v1/admin/notifications/broadcast    → send to all users (admin)
POST   /v1/admin/notifications/test-email   → synchronous diagnostic send (admin)
```

- Push is enabled by setting `FIREBASE_CREDENTIALS_FILE`; if unset, FCM
  delivery is skipped and only the in-app feed + email channel are used.
- Per-user preferences control which channels (in-app/email/push) a
  notification type is delivered through.
- Every send/broadcast/fail logged as outbound activity, same as email/SMS.

---

### 8. App Versioning System

Protects APIs from outdated mobile clients.

**How it works:**
1. Client sends `X-App-Version: 2.4.1` and `X-App-Platform: ios` on every request
2. `VersionCheck` middleware compares against the stored policy
3. Returns verdict in `X-Version-Status` response header
4. If `forceUpdate=true` and client is below `minVersion` → **HTTP 426** (blocks request)

```
GET  /v1/app-version/check              → public version check
GET  /v1/admin/app-versions             → list all policies (admin)
PUT  /v1/admin/app-versions/:platform   → update policy (admin)
```

Platforms: `ios` | `android` | `web`  
Status values: `up_to_date` | `available` | `required`

---

### 9. Background Job Queue Backends (Redis / RabbitMQ / Kafka)

Background jobs run on one of three interchangeable transports, selected by
`QUEUE_DRIVER` in `.env`. All three expose the same enqueue-side contract
(`EnqueueEmail`), so switching backends never requires touching the code that
enqueues a job — only which worker process you run.

| `QUEUE_DRIVER` | Worker | Start with |
|---|---|---|
| `redis` (default) | `apps/worker` (Asynq) | `make docker-worker` / `make run-worker` / `make dev-worker` |
| `rabbitmq` | `apps/worker-rabbitmq` | `make docker-rabbitmq` / `make run-worker-rabbitmq` / `make dev-worker-rabbitmq` |
| `kafka` | `apps/worker-kafka` | `make docker-kafka` / `make run-worker-kafka` / `make dev-worker-kafka` |

**`QUEUE_DRIVER` must match the worker you start** — the API only publishes
jobs, it never consumes them. A mismatch means jobs queue up on a broker
nobody is listening to, with no error raised.

```bash
# RabbitMQ — management UI at http://localhost:15672 (guest/guest)
make docker-rabbitmq

# Kafka — broker reachable at localhost:9094 from the host, kafka:9092 in-network
make docker-kafka

# First time using RabbitMQ/Kafka? Pull in their pure-Go client libraries:
make deps
```

`pkg/queue` defines the transport-neutral `Publisher`/`Consumer` contract that
`pkg/queue/rabbitmq` and `pkg/queue/kafka` implement, mirroring `pkg/worker`'s
Asynq client/server shape. Task handlers in `pkg/queue/tasks` are shared
unchanged by both `apps/worker-rabbitmq` and `apps/worker-kafka`.

#### Routing different workloads to different brokers

`QUEUE_DRIVER` is the default backend for everything. Two optional overrides
route specific task types to a *different* backend at the same time, instead
of picking one broker for the whole app — e.g. RabbitMQ for
reliability-sensitive transactional email, Kafka for high-throughput
promotional/bulk email:

```env
QUEUE_TRANSACTIONAL_DRIVER=rabbitmq   # welcome/verify/reset/OTP email + admin "send to one user"
QUEUE_PROMOTIONAL_DRIVER=kafka        # NotificationService.Broadcast's email channel
```

Both are blank by default (falls back to `QUEUE_DRIVER` — today's
single-backend behavior, unchanged unless you opt in). `pkg/queue/router`
implements this: it builds one `queue.Publisher` per distinct driver
referenced and wires them into a `Router` that itself satisfies
`queue.Publisher`, so no caller needs to know routing is happening. Every
worker still registers the same handler for both `email:send` (transactional)
and `email:send:promotional` (bulk) — routing only decides which broker a job
lands on, not how it's consumed, so **every broker referenced by either
override needs its own worker process running**:

```bash
make docker-queues   # starts Redis + RabbitMQ + Kafka workers all at once —
                      # needed to actually exercise a split like the example above
```

(`docker-worker`/`docker-rabbitmq`/`docker-kafka` individually only start
one broker+worker pair each, which is enough when everything uses the same
`QUEUE_DRIVER`.)

---

### 10. Web Test Client

`web/` is a React + TypeScript SPA (Vite, Tailwind, React Router) that
exercises every HTTP-exposed endpoint of `apps/api` — auth, 2FA, passkeys
(real WebAuthn ceremonies via `@simplewebauthn/browser`), mobile OTP,
notifications, admin users/roles/app-versions/analytics/notifications,
health — plus a generic raw-request API console, an **i18n Compare** page
(`/i18n-compare`) that fires one request twice with different
`x-custom-lang` headers to compare translated responses side by side, and a
client-side activity log. It talks to the API directly over `fetch` (CORS
already allows `*`); no backend changes are needed to support it.

**Admin → Notifications** (`/admin/notifications`) is the page for manually
exercising queue routing (see "Routing different workloads to different
brokers" above): a **Test email** form for a quick `MAIL_DRIVER` sanity
check, a **Send to one user** form, and a **Broadcast** form with a
user-checkbox picker — broadcasting to the `email` channel is what actually
triggers `QUEUE_PROMOTIONAL_DRIVER`. Create a user from **Admin → Users**
first (fires the transactional welcome email), then broadcast to `email` and
compare which worker log picks up each one.

```bash
make web-install     # cd web && npm install
make web-dev          # cd web && vite dev → http://localhost:5173
```

Settings (API base URL, `x-custom-lang`, `X-Tenant-ID`, `X-App-Version`,
`X-App-Platform`) are editable at runtime from the app's own **Settings**
page and persisted in `localStorage` — no rebuild needed to point it at a
different API instance. A 401 on any authenticated request first tries a
silent `/v1/auth/refresh`; only a failed refresh logs the session out.

**WebAuthn origin:** `WEBAUTHN_RP_ORIGIN` must equal the exact origin that
calls `navigator.credentials.*` in the browser — `web/` at
`http://localhost:5173`, not the API's own `:3000`.

See [web/README.md](web/README.md) for the full breakdown.

### 11. Dev Container (VS Code / GitHub Codespaces)

`.devcontainer/` ships a fully configured environment — Go 1.23, Node 20,
`air`, `golangci-lint`, `goimports`, `mongosh`, Docker CLI — with MongoDB rs0
and Redis started automatically. `post-create.sh` generates JWT keys,
patches `.env` with in-network service URIs, builds the React Email
templates, generates Swagger docs, and runs `go mod tidy` on first open.
Works identically in GitHub Codespaces. VS Code Tasks and an F5 debug
launch config are included for the API server.

---

## Updated Route Table

```
POST   /v1/auth/login                          public
POST   /v1/auth/register                       public
POST   /v1/auth/refresh                        public
DELETE /v1/auth/logout                         Bearer
DELETE /v1/auth/logout-all                     Bearer
POST   /v1/auth/2fa/enable                     Bearer
POST   /v1/auth/2fa/verify                     Bearer
GET    /v1/auth/github                         public  → redirect
GET    /v1/auth/github/callback                public  ← GitHub redirect
POST   /v1/auth/apple/callback                 public
POST   /v1/auth/passkey/login/begin            public  (known gap — see Passkey section)
POST   /v1/auth/passkey/login/finish           public  (known gap — see Passkey section)
POST   /v1/auth/passkey/identified/begin       public  (known gap — see Passkey section)
POST   /v1/auth/passkey/identified/finish      public  (known gap — see Passkey section)

GET    /v1/me                                  Bearer
PATCH  /v1/me                                  Bearer
PATCH  /v1/me/password                         Bearer
POST   /v1/me/passkeys/register/begin          Bearer
POST   /v1/me/passkeys/register/finish         Bearer
GET    /v1/me/passkeys                         Bearer
DELETE /v1/me/passkeys/:id                     Bearer
POST   /v1/me/mobiles/send-otp                 Bearer
POST   /v1/me/mobiles/verify                   Bearer
GET    /v1/me/mobiles                          Bearer
PATCH  /v1/me/mobiles/:e164/primary            Bearer
DELETE /v1/me/mobiles/:e164                    Bearer
GET    /v1/me/notifications                    Bearer
GET    /v1/me/notifications/unread-count       Bearer
PATCH  /v1/me/notifications/:id/read           Bearer
PATCH  /v1/me/notifications/read-all           Bearer
GET    /v1/me/notifications/preferences        Bearer
PATCH  /v1/me/notifications/preferences        Bearer
POST   /v1/me/notifications/devices            Bearer
DELETE /v1/me/notifications/devices/:token     Bearer

GET    /v1/app-version/check                   public

GET    /v1/roles                               Admin

GET    /v1/users                               Admin
POST   /v1/users                               Admin
GET    /v1/users/:id                           Admin
PATCH  /v1/users/:id                           Admin
DELETE /v1/users/:id                           Admin
POST   /v1/users/:id/block                     Admin
POST   /v1/users/:id/unblock                   Admin

GET    /v1/admin/app-versions                  Admin
PUT    /v1/admin/app-versions/:platform        Admin

POST   /v1/admin/notifications/send            Admin
POST   /v1/admin/notifications/broadcast       Admin
POST   /v1/admin/notifications/test-email      Admin

GET    /v1/admin/analytics/...                 Admin  (all analytics routes)

GET    /health                                 public
GET    /health/live                            public
GET    /health/ready                           public
```
