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
| Background jobs | BullMQ | **asynq** (Redis, default) — or **RabbitMQ** / **Kafka** via `QUEUE_DRIVER` |
| Config | @nestjs/config | **viper** |
| Validation | class-validator | **go-playground/validator** |
| Logging | Winston | **zap** (Uber) |
| API docs | Swagger (NestJS) | **swaggo/swag** |
| Error tracking | Sentry | **sentry-go** |
| Email | AWS SES + React Email | **aws-sdk-go-v2 SES** + **React Email** (compiled to embedded HTML) |
| File storage | AWS S3 | **aws-sdk-go-v2 S3** |
| OAuth | passport-google | **golang.org/x/oauth2** |
| Hot reload | ts-node-dev | **air** |
| Containerisation | Docker | **Docker** multi-stage |

---

## Features

### 🎯 Architecture
- **Repository Pattern** — clean data access abstraction (no ORM lock-in)
- **Modular structure** — `internal/<module>/{entity,dto,repository,service,handler}`
- **SOLID principles** — interfaces everywhere, easy to mock & test
- **12-Factor App** compliant config via environment variables

### 🔐 Authentication & Security
- JWT **ES256** access tokens + **ES512** refresh tokens (asymmetric keys)
- **Stateful sessions** — Redis-backed with per-session revocation
- **Token rotation** — new pair issued on every refresh
- **TOTP 2FA** with backup codes
- **RBAC** — role + permission checks via middleware decorators
- **API Key** auth (hashed with bcrypt, prefix-indexed lookup)
- **Rate limiting** — fixed-window counter per IP via Redis
- **Brute-force protection** — automatic account lockout
- CORS, Gzip, security headers

### 📊 Database & Storage
- MongoDB with **transaction support** (requires replica set)
- **TTL indexes** on activity logs (auto-expiry after 90 days)
- Redis **multi-level caching** (feature flags, sessions)
- **AWS S3** — upload, presigned upload/download, delete
- **Soft deletes** on users

### ⚡ Performance
- Background job queue (email, push, cleanup tasks) — **Redis/Asynq** by default, swappable for **RabbitMQ** or **Kafka** via `QUEUE_DRIVER` (see "Background Job Queue Backends" below)
- Response **gzip compression**
- **Cursor + offset** pagination with metadata
- Per-collection MongoDB indexes

### 🛠 Developer Experience
- **Hot reload** via `air`
- `make setup` one-command bootstrap
- `make generate-keys` — EC key pair generation
- `make swagger` — Swagger docs generation
- `make seed` — database seeding (roles, users, flags)
- `make test-coverage` — HTML coverage report
- golangci-lint config

### 📡 Integrations
- **Sentry** error tracking + traces
- **AWS SES** transactional email (HTML templates)
- **Activity logging** with 90-day auto-expiry
- **Feature flags** (MongoDB + Redis cache)
- **Health checks** — `/health`, `/health/live`, `/health/ready`

---

## Project Structure

```
bread-golang-boilerplate/
├── cmd/api/                    # Entry point (main.go)
├── internal/
│   ├── app/                    # Gin bootstrap, dependency wiring
│   ├── config/                 # Viper config loader
│   ├── database/               # MongoDB + Redis clients
│   ├── common/
│   │   ├── errors/             # Domain error types + sentinels
│   │   ├── middleware/         # Auth, API Key, Rate limit, Logger, Feature flag
│   │   ├── pagination/         # Offset + cursor helpers
│   │   └── response/           # Standard JSON envelope
│   ├── auth/                   # Login, Register, Refresh, Logout, 2FA
│   │   ├── dto/
│   │   ├── handler/
│   │   └── service/
│   ├── user/                   # CRUD, block/unblock, password change
│   │   ├── dto/
│   │   ├── entity/
│   │   ├── handler/
│   │   ├── repository/
│   │   └── service/
│   ├── role/                   # RBAC roles + permissions
│   ├── apikey/                 # API key management
│   ├── activity/               # Audit log
│   ├── featureflag/            # Feature flags with Redis cache
│   └── health/                 # Liveness + readiness probes
├── pkg/
│   ├── jwt/                    # ES256/ES512 token manager
│   ├── hash/                   # bcrypt wrapper
│   ├── logger/                 # Zap factory
│   ├── email/                  # AWS SES mailer
│   ├── storage/                # AWS S3 client
│   ├── worker/                 # Asynq (Redis) client + server + task handlers
│   └── queue/                  # Backend-agnostic Publisher/Consumer contract
│       ├── rabbitmq/           # RabbitMQ (AMQP) implementation
│       ├── kafka/               # Kafka implementation
│       └── tasks/               # Shared handlers for the RabbitMQ/Kafka workers
├── scripts/
│   ├── seed/                   # Seed roles, users, feature flags
│   └── migrate/                # Create MongoDB indexes
├── keys/                       # EC key pairs (git-ignored)
├── .env.example
├── .air.toml
├── .golangci.yml
├── docker-compose.yml
├── Dockerfile
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

# Start with hot reload
make dev
```

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

### Health
| Method | Path |
|--------|------|
| GET | `/health` |
| GET | `/health/live` |
| GET | `/health/ready` |

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

---

## Middleware Stack (in order)

```
RequestID → Recovery → Logger → Gzip → CORS → RateLimit
  → APIKeyProtected (optional)
  → AuthJWTAccess
  → RoleProtected(roles...)
  → FeatureFlagProtected(flagKey)
  → Handler
```

Mirrors the NestJS decorator ordering from the original boilerplate.

---

## Testing

```bash
make test              # all tests
make test-unit         # short tests only
make test-coverage     # HTML coverage report
```

---

## Environment Variables

See [`.env.example`](.env.example) for the full list with descriptions.

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

---

### 4. GitHub SSO

Full OAuth2 flow with CSRF state protection:

```
GET  /v1/auth/github           →  redirects to GitHub
GET  /v1/auth/github/callback  →  exchanges code, auto-registers or links account, returns token pair
```

- State token stored in Redis (10-min TTL)  
- Fetches primary verified email from `/user/emails` if profile email is private  
- Auto-links to existing account if email matches  

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
| `redis` (default) | `apps/worker` (Asynq) | `make docker-worker` / `make run-worker` |
| `rabbitmq` | `apps/worker-rabbitmq` | `make docker-rabbitmq` / `make run-worker-rabbitmq` |
| `kafka` | `apps/worker-kafka` | `make docker-kafka` / `make run-worker-kafka` |

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

---

## Updated Route Table

```
POST   /v1/auth/login                    public
POST   /v1/auth/register                 public
POST   /v1/auth/refresh                  public
DELETE /v1/auth/logout                   Bearer
DELETE /v1/auth/logout-all               Bearer
POST   /v1/auth/2fa/enable               Bearer
POST   /v1/auth/2fa/verify               Bearer
GET    /v1/auth/github                   public  → redirect
GET    /v1/auth/github/callback          public  ← GitHub redirect
POST   /v1/auth/passkey/login/begin      public
POST   /v1/auth/passkey/login/finish     public
POST   /v1/auth/passkey/identified/begin public
POST   /v1/auth/passkey/identified/finish public

GET    /v1/me                            Bearer
PATCH  /v1/me                            Bearer
PATCH  /v1/me/password                   Bearer
POST   /v1/me/passkeys/register/begin    Bearer
POST   /v1/me/passkeys/register/finish   Bearer
GET    /v1/me/passkeys                   Bearer
DELETE /v1/me/passkeys/:id               Bearer
POST   /v1/me/mobiles/send-otp           Bearer
POST   /v1/me/mobiles/verify             Bearer
GET    /v1/me/mobiles                    Bearer
PATCH  /v1/me/mobiles/:e164/primary      Bearer
DELETE /v1/me/mobiles/:e164              Bearer

GET    /v1/app-version/check             public

GET    /v1/users                         Admin
POST   /v1/users                         Admin
GET    /v1/users/:id                     Admin
PATCH  /v1/users/:id                     Admin
DELETE /v1/users/:id                     Admin
POST   /v1/users/:id/block               Admin
POST   /v1/users/:id/unblock             Admin

GET    /v1/admin/app-versions            Admin
PUT    /v1/admin/app-versions/:platform  Admin

GET    /v1/admin/analytics/...           Admin  (all analytics routes)
```
