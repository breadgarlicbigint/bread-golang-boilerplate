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
make dev                # hot-reload with air (also regenerates swagger + rebuilds changed email templates)
make test               # all unit tests
make test-coverage      # HTML coverage report → coverage.html
make swagger            # regenerate Swagger docs (uses go run, no install needed)
make build-emails       # compile React Email .tsx → pkg/email/dist/ (requires Node.js)
make web-install        # install web/ (React test client) dependencies
make web-dev            # run the web test client → http://localhost:5173
make lint               # golangci-lint
make deps               # go get the RabbitMQ + Kafka client libs and sync go.sum
make run-worker          # build + run the Redis/Asynq worker (apps/worker)
make run-worker-rabbitmq # build + run the RabbitMQ worker (apps/worker-rabbitmq)
make run-worker-kafka    # build + run the Kafka worker (apps/worker-kafka)
make dev-worker          # hot-reload the Redis/Asynq worker with air
make dev-worker-rabbitmq # hot-reload the RabbitMQ worker with air
make dev-worker-kafka    # hot-reload the Kafka worker with air
make docker-worker       # docker-up + Redis/Asynq worker (profile: worker)
make docker-rabbitmq     # docker-up + RabbitMQ broker + worker (profile: rabbitmq)
make docker-kafka        # docker-up + Kafka broker + worker (profile: kafka)
make docker-mqtt         # docker-up + Mosquitto MQTT broker (modules/iot demo, profile: mqtt)
make docker-monitoring   # docker-up + Prometheus + Grafana (profile: monitoring)
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
| Background jobs | `hibiken/asynq` (Redis, default) or `rabbitmq/amqp091-go` / `segmentio/kafka-go` — selected via `QUEUE_DRIVER` (`pkg/worker`, `pkg/queue`) |
| Realtime (WS/SSE) | `gorilla/websocket` + stdlib SSE (`pkg/realtime`, `modules/realtime`) — see "Realtime (WebSocket / SSE / Pub-Sub)" |
| MQTT | `eclipse/paho.mqtt.golang` (`pkg/mqtt`, `modules/iot` device-telemetry demo) — see "IoT (MQTT Device Telemetry Demo)" |
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
| Metrics | `prometheus/client_golang` (`pkg/metrics`) — HTTP + MongoDB + Go runtime metrics, scraped by Prometheus, visualized in Grafana — see "Monitoring (Prometheus / Grafana)" |

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
    main.go                     ← Asynq (Redis) background worker entry point
  worker-rabbitmq/
    main.go                     ← RabbitMQ background worker entry point — same
                                   task handlers as worker/, wired via pkg/queue
  worker-kafka/
    main.go                     ← Kafka background worker entry point — same
                                   task handlers as worker/, wired via pkg/queue

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
  iot/                         ← MQTT device-telemetry demo (pkg/mqtt)
  mobile/
  notification/
  passkey/
  realtime/                    ← WebSocket/SSE + generic pub-sub (pkg/realtime)
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
  metrics/                      ← Prometheus registry: HTTP Gin middleware,
                                   MongoDB CommandMonitor, /metrics handler
  mqtt/                         ← paho MQTT client wrapper (modules/iot)
  realtime/                     ← WebSocket/SSE fan-out Hub (modules/realtime)
  sms/                          ← Twilio SMS + WhatsApp sender
  storage/                      ← AWS S3 presign / upload / delete
  uuidbson/                     ← UUID BSON Binary-4 codec
  worker/                       ← Asynq (Redis) client + server + task handlers
  queue/                        ← Backend-agnostic Publisher/Consumer contract
    rabbitmq/                   ← RabbitMQ (AMQP) Publisher/Consumer impl
    kafka/                      ← Kafka Publisher/Consumer impl
    tasks/                      ← Transport-agnostic job handlers shared by
                                   worker-rabbitmq and worker-kafka

locales/                        ← en.json, id.json (i18n strings incl. email tokens)
email-templates/                ← React Email .tsx source (build-time, Node.js only)
monitoring/                     ← Prometheus scrape config + Grafana provisioning
                                   (datasource + dashboard) — see "Monitoring
                                   (Prometheus / Grafana)" below
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
result, err := h.svc.Create(c.Request.Context(), req)
if err != nil {
    response.HandleAppError(c, err)
    return
}
```

`response.HandleAppError` (`shared/response/response_i18n.go`) is the single
translation point from a service-layer error to an HTTP response: it unwraps
`*errors.AppError`, translates via `ae.Key` (falling back to the raw
`ae.Message` if `Key` is empty), and — for anything that isn't an `AppError`
— logs the real error server-side and returns a generic translated 500. Call
it directly; don't write a per-handler `handleError` wrapper (older handlers
had one — they've since been consolidated onto this shared helper).

Never expose raw error strings to the client. All domain errors go through
`shared/errors.AppError` which carries a status code and safe message. Build
sentinels with `errors.NewI18n(status, code, key, message)` — `key` is the
`locales/*.json` key translated for the response; use `errors.New(status,
code, message)` (no key) only for messages that genuinely have no
translation and shouldn't get one.

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

**Every handler response goes through the i18n-aware helpers** —
`response.OKI18n`, `response.CreatedI18n`, `response.OKWithMetaI18n`,
`response.ErrorI18n`, `response.UnauthorizedI18n`, `response.ForbiddenI18n`,
`response.NotFoundI18n` — each takes a `locales/*.json` key instead of a
literal string, e.g. `response.OKI18n(c, "user.createSuccess", data)` instead
of `response.OK(c, "User created", data)`. The non-`I18n` variants
(`response.OK`, `response.Error`, ...) still exist and are what the `I18n`
helpers call internally, but a handler should never hardcode a client-facing
message string directly — add a locale key instead (see "Multi-language"
below). The one deliberate exception is
`POST /v1/admin/notifications/test-email`, which echoes a raw transport
`err.Error()` into the response **data** (not the translated `message`) by
design — see "Async delivery via the worker queue".

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
        response.HandleAppError(c, err)
        return
    }
    response.CreatedI18n(c, "xxx.createSuccess", mapToResponse(result))
}
```

Add `"createSuccess": "Xxx created"` under a new `xxx` namespace in both
`locales/en.json` and `locales/id.json` (see "Multi-language" below).

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

### Async delivery via the worker queue (transactional emails)

Transactional emails triggered by an HTTP request must NOT block the request on
a slow/failure-prone SMTP/SES round-trip. The pattern is **render in the API,
send in the worker**:

```
API request path                          Worker process (apps/worker)
────────────────                          ────────────────────────────
localMailer.RenderWelcome(...)  ┐
localMailer.SubjectWelcome(...) ├─ fast, in-memory (i18n needs request `lang`)
                                ┘
jobQueue.EnqueueEmail(to,subject,html,text)  ──►  Redis (asynq "default" queue)
                                                        │
                                                        ▼
                                             EmailTaskHandler.Handle
                                             mailer.Send(...)  ← retryable (MaxRetry 5)
```

- `worker.Client.EnqueueEmail(to, subject, html, text)` is the enqueue entry
  point. Services depend on the small `EmailQueue` interface (see
  `modules/user/service/user.service.go`), never on asynq directly.
- The API wires a `*worker.Client` in `apps/api/app/app.go` (`jobQueue`) and
  passes it into services; it is closed in `App.Shutdown`.
- The reference implementation is the **welcome email on admin user creation**
  (`UserService.Create` → `sendWelcomeEmail` → `jobQueue.EnqueueEmail`). Follow
  it for any new request-triggered transactional email.
- **The worker process must be running for these emails to actually be sent.**
  `make dev` (API only) just enqueues to Redis — jobs sit in the queue until a
  worker consumes them. Run `make docker-worker` (or `make run-worker` /
  `make dev-worker` for hot reload) too.
- **Diagnostic exception:** `POST /v1/admin/notifications/test-email` sends
  synchronously via `NotificationService.SendTestEmail` and returns the raw
  transport error (`{"sent":false,"error":"..."}`), specifically so mail-driver
  config can be verified without a worker running. Keep genuinely diagnostic /
  "need the result now" sends synchronous; everything else goes through the queue.

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

**Under `make dev`, step 6 happens automatically:** `scripts/dev/air-build-api.sh`
(the air build step) checks whether any file under `email-templates/src` is
newer than `email-templates/.build-stamp` (a marker touched after the last
build) and, only if so, runs `npm install`/`npm run build` before the Go
build — so editing a `.tsx` template and saving is enough to pick up the
change on the next hot reload, no separate `make build-emails` needed. This
only runs for `make dev` (`.air.toml`); the worker air configs
(`.air.worker*.toml`) don't render templates so they skip it. Delete
`email-templates/.build-stamp` to force a rebuild on the next save.

---

## Background Job Queue Backends (Redis / RabbitMQ / Kafka)

Background jobs (currently: transactional email delivery, with notification
send/broadcast handlers also wired) can run on three interchangeable
transports, selected by `QUEUE_DRIVER`:

| `QUEUE_DRIVER` | Worker process | Enqueue-side package | Consume-side package |
|---|---|---|---|
| `redis` (default) | `apps/worker` | `pkg/worker` (`worker.Client`) | `pkg/worker` + `pkg/worker/tasks` |
| `rabbitmq` | `apps/worker-rabbitmq` | `pkg/queue/rabbitmq` (`rabbitmq.Publisher`) | `pkg/queue/rabbitmq` + `pkg/queue/tasks` |
| `kafka` | `apps/worker-kafka` | `pkg/queue/kafka` (`kafka.Publisher`) | `pkg/queue/kafka` + `pkg/queue/tasks` |

`pkg/queue/queue.go` defines the backend-agnostic contract (`Publisher`,
`Consumer`, `Delivery`, `HandlerFunc`) that the RabbitMQ and Kafka packages
implement — it mirrors the shape of the Redis/Asynq implementation in
`pkg/worker` so all three are drop-in parallel. Task-type strings
(`email:send`, `email:send:promotional`, `notification:send`,
`notification:broadcast`, `session:cleanup`) and JSON payload shapes are
identical across all three, so switching `QUEUE_DRIVER` doesn't change what a
service enqueues.

**`QUEUE_DRIVER` MUST match the worker process you actually run** — the API
only publishes jobs, it never consumes them. `apps/api/app/app.go`'s
`buildJobQueue` picks the `Publisher` at startup from `QUEUE_DRIVER`; a
mismatch (e.g. `QUEUE_DRIVER=redis` while only `apps/worker-rabbitmq` is
running) means jobs are published to a broker nobody consumes, so they queue
up and never run — no error, no crash, just silence.

```bash
# Redis (Asynq) — default
make docker-worker       # or: make run-worker      # or: make dev-worker (hot reload)

# RabbitMQ — management UI at http://localhost:15672 (guest/guest)
make docker-rabbitmq     # or: make run-worker-rabbitmq  # or: make dev-worker-rabbitmq

# Kafka — broker at localhost:9094 (host) / kafka:9092 (in-network)
make docker-kafka        # or: make run-worker-kafka     # or: make dev-worker-kafka
```

### Per-workload routing (`pkg/queue/router`)

`QUEUE_DRIVER` picks the *default* backend for every task type. Two optional
overrides let different workloads ride different brokers at the same time,
instead of picking one broker for everything:

```env
QUEUE_TRANSACTIONAL_DRIVER=rabbitmq   # queue.TaskSendEmail — reliability-sensitive
QUEUE_PROMOTIONAL_DRIVER=kafka        # queue.TaskSendPromotionalEmail — throughput-sensitive
```

Both are blank by default, meaning "fall back to `QUEUE_DRIVER`" — today's
single-backend behavior is unchanged unless you opt in. `apps/api/app/app.go`'s
`buildJobQueue` builds a `queue.Publisher` for each distinct driver referenced
(`buildPublisher`) and wires them into a `*router.Router`, which itself
satisfies `queue.Publisher` and is what `App.jobQueue` actually holds — every
caller (`EmailQueue`, `TransactionalEmailQueue`, `PromotionalEmailQueue`) is
unaware routing is happening. If a transactional/promotional override fails
to connect, it logs a warning and falls back to the default driver rather
than disabling jobs outright; only a default-driver failure disables jobs
entirely (`nil`).

- `EnqueueEmail` publishes `queue.TaskSendEmail` → routed by
  `QUEUE_TRANSACTIONAL_DRIVER`. Callers: the welcome/verify/reset/OTP flows
  (`modules/user/service`, `modules/auth/service`), and
  `NotificationService.Send`'s email channel
  (`modules/notification/service`) — a single admin-triggered notification is
  treated the same as those flows (reliability-sensitive, low-volume). Falls
  back to a synchronous send when no transactional queue is configured.
- `EnqueuePromotionalEmail` publishes `queue.TaskSendPromotionalEmail` →
  routed by `QUEUE_PROMOTIONAL_DRIVER`. The one real caller today is
  `NotificationService.Broadcast`'s email channel — when a promotional queue
  is configured, broadcasting to many users enqueues one job per recipient
  instead of blocking the HTTP request on N sequential SMTP/SES sends;
  `success`/`failed` in the response then mean "queued", not "delivered".
  Without a promotional queue configured, `Broadcast` keeps the original
  synchronous per-user path.
- `apps/api/app/app.go` passes the same `*router.Router` (`jobQueue`) as both
  the `TransactionalEmailQueue` and `PromotionalEmailQueue` arguments to
  `notifSvc.New` — the router itself resolves each call to the right backend
  by task type, so a single `Router` instance correctly backs both roles.
- Both task types share one `EmailTaskHandler` on every worker — routing only
  changes which broker a job lands on, not how it's consumed. **Whichever
  broker(s) you route to must have a worker actually running against them**,
  same rule as `QUEUE_DRIVER` — `make docker-queues` starts all three
  (Redis + RabbitMQ + Kafka workers) at once for exercising a split like the
  example above; `make docker-worker`/`docker-rabbitmq`/`docker-kafka`
  individually only start one.

`make deps` runs `go get` for the RabbitMQ (`rabbitmq/amqp091-go`) and Kafka
(`segmentio/kafka-go`) client libraries and syncs `go.sum` — both are
pure-Go clients (no CGO/librdkafka), so `CGO_ENABLED=0` builds still work.

### Hot reload for workers (local dev)

Like `make dev` for the API, each worker has an air-backed hot-reload target
that rebuilds and restarts on `.go` file changes — run whichever one matches
`QUEUE_DRIVER` in `.env`, against a locally reachable Redis/RabbitMQ/Kafka +
MongoDB (e.g. via `make docker-up`, which starts Mongo + Redis but not the
worker container itself):

```bash
make dev-worker            # Redis/Asynq worker, air config: .air.worker.toml
make dev-worker-rabbitmq   # RabbitMQ worker,    air config: .air.worker-rabbitmq.toml
make dev-worker-kafka      # Kafka worker,       air config: .air.worker-kafka.toml
```

Each target's tmp/build dir (`.air-worker/`, `.air-worker-rabbitmq/`,
`.air-worker-kafka/`) is separate from `.air/` (used by `make dev`) and from
each other, so `make dev`, `make dev-worker`, and one of the queue-specific
worker targets can all run at once without clobbering each other's binaries.

Services depend only on the small `EmailQueue` interface
(`EnqueueEmail(to, subject, html, text) error`) — never on `asynq`, `amqp091-go`,
or `kafka-go` directly — so adding a fourth transport means implementing
`queue.Publisher`/`queue.Consumer` and adding one case to `buildJobQueue`; no
caller changes.

Handlers for the RabbitMQ/Kafka workers live in `pkg/queue/tasks` (the
parallel of `pkg/worker/tasks`) and are registered identically by both
`apps/worker-rabbitmq/main.go` and `apps/worker-kafka/main.go` via
`tasks.RegisterAll` / `tasks.RegisterNotifications`.

---

## Realtime (WebSocket / SSE / Pub-Sub)

`pkg/realtime` is a transport-agnostic, in-process pub/sub fan-out core
(`Hub` — `Subscribe`/`Unsubscribe`/`Publish`/`Stats`, zero domain knowledge).
`modules/realtime` wraps it with the one domain convention other code needs
— every user has a private topic `user:<id>` — and exposes it over HTTP:

```
GET  /v1/me/ws                    → upgrades to WebSocket, auto-subscribed to the caller's private topic
GET  /v1/me/events                → SSE stream, defaults to the caller's private topic; ?topic= to watch another
POST /v1/admin/realtime/publish   → admin: publish an arbitrary event to an arbitrary topic (generic pub/sub test)
GET  /v1/admin/realtime/stats     → admin: current topic/connection counts
```

- **Auth over WebSocket/SSE:** browsers' native `WebSocket` and `EventSource`
  APIs cannot set an `Authorization` header. `middleware.AuthJWTAccessWS`
  (used only on `/v1/me/ws` and `/v1/me/events`) accepts the token via
  `?token=` as a fallback — `middleware.AuthJWTAccess` (header-only) still
  guards every other route, including the admin realtime endpoints above.
- **Generic pub/sub over WebSocket:** after connecting, a client may send
  `{"action":"subscribe","topic":"..."}` / `{"action":"unsubscribe",...}` JSON
  frames to join/leave additional topics beyond its auto-subscribed private
  channel — see `modules/realtime/handler/realtime.handler.go`'s
  `wsReadPump`. SSE is one-directional, so its topic is fixed at connect
  time via `?topic=`.
- **Notification integration:** `NotificationService.Send` (`modules/notification/service`)
  pushes a live event to `PublishToUser` on top of (not instead of) its
  normal channel-specific delivery (email/push/in-app/silent) — every
  `POST /v1/admin/notifications/send` or `/broadcast` call is visible on a
  connected WebSocket/SSE client in addition to being persisted for
  `GET /v1/me/notifications`. `modules/notification/service` depends on a
  small local `RealtimePublisher` interface (not the full `realtime.Service`
  contract) — `nil` (e.g. inside a worker process, which holds no
  connections) just skips the live push.
- **Fire-and-forget, no replay:** if nobody is connected to a topic when
  `Publish` is called, the event is silently dropped — there is no buffering.
  `GET /v1/me/notifications` and `GET /v1/admin/iot/telemetry` are the
  durable records; realtime delivery is "while you're watching" on top.
- **Web test client:** `/realtime` (`web/src/pages/RealtimePage.tsx`) —
  connect/disconnect panels for both WebSocket and SSE with a live event
  log, a join/leave-topic control for the WebSocket panel, the admin publish
  form, and a stats panel. This is the page to use to manually verify any of
  the above.

## IoT (MQTT Device Telemetry Demo)

`pkg/mqtt` is a thin wrapper around `eclipse/paho.mqtt.golang` (Connect/
Publish/Subscribe/Close). `modules/iot` is a self-contained demo built on
it — **not** a fourth `QUEUE_DRIVER` backend, unrelated to the task-queue
system above:

```
POST /v1/admin/iot/devices/:deviceId/simulate  → publishes to MQTT topic devices/:deviceId/telemetry, as if a real device sent it
POST /v1/admin/iot/devices/:deviceId/command   → publishes to devices/:deviceId/commands (fire-and-forget; nothing subscribes to it — a real device would)
GET  /v1/admin/iot/telemetry                   → paginated, persisted readings (MongoDB, TTL 1 day)
```

The full round trip: `Simulate` → MQTT broker → a subscriber on
`devices/+/telemetry` running inside the API process (`IoTService.Start`,
registered once at startup) → persists to the `device_telemetry` collection
→ forwards the reading onto `modules/realtime`'s generic pub/sub topic
`iot:telemetry` (via the same narrow `RealtimePublisher` pattern
`modules/notification` uses) → any WebSocket/SSE client watching that topic
sees it arrive live. Connect the **Realtime** page (WS or SSE, topic
`iot:telemetry`) and then click **Simulate** on the **IoT** page
(`/admin/iot`) to watch the whole chain fire.

- **Optional, like Firebase/Apple/GitHub OAuth:** `MQTT_BROKER_URL` empty
  (the default) or a failed dial at startup just disables `modules/iot` —
  every call returns `errors.ErrMQTTNotConfigured` (503) rather than
  breaking API startup. `apps/api/app/app.go` logs a warning and continues.
- **Broker:** `make docker-mqtt` starts an Eclipse Mosquitto container
  (`mosquitto/mosquitto.conf` — anonymous access, local-dev only, no TLS) on
  the `mqtt` Compose profile. Set `MQTT_BROKER_URL=tcp://mosquitto:1883` in
  `.env` first, same "the driver you pick must have its backing service
  actually running" rule as `QUEUE_DRIVER`.
- Services depend on the small `MQTTClient` interface (`Publish`/`Subscribe`)
  in `modules/iot/service`, not `pkg/mqtt` directly — a `*pkg/mqtt.Client`
  satisfies it; tests fake just the `Publisher` half (no real broker).

---

## Monitoring (Prometheus / Grafana)

`pkg/metrics` wraps `prometheus/client_golang` and registers on the default
Prometheus registry — so `GET /metrics` (always on, unauthenticated, same
tier as `/health`) exposes three things in one scrape:

1. **HTTP request metrics** — `http_requests_total{method,path,status}` and
   `http_request_duration_seconds{method,path,status}` (histogram), recorded
   by `metrics.GinMiddleware()` (wired into `apps/api/app/app.go`'s global
   `engine.Use(...)` chain) for every request except `/metrics` itself
   (excluded so scrapes don't inflate their own series). `path` is Gin's
   route pattern (`c.FullPath()`, e.g. `/v1/users/:id`), not the raw URL, so
   cardinality stays bounded; unmatched routes (404s with no registered
   handler) label as `path="unmatched"`.
2. **MongoDB command metrics + logs** — `mongodb_operations_total{operation,
   collection,status}` and `mongodb_operation_duration_seconds{operation,
   collection}` (histogram), plus a zap `Debug`/`Warn` log line per command,
   recorded by `metrics.MongoCommandMonitor(log)`, an
   `*event.CommandMonitor` passed to `database.NewMongoDBWithMonitor` (only
   `apps/api/main.go` opts in — `database.NewMongoDB` still exists unchanged
   and is what `scripts/seed`, `scripts/migrate`, and the worker `main.go`s
   keep calling, since one-shot scripts and workers weren't in scope for
   this pass). High-frequency driver handshake/keepalive commands (`hello`,
   `ismaster`, `ping`, `saslStart`, `saslContinue`, `endSessions`,
   `buildInfo`, `getLastError`) are filtered out in `Started` before they're
   ever tracked, so they inflate neither the logs nor the metric
   cardinality. The collection name is only present on the `Started` event's
   raw command document (e.g. `{"find": "users", ...}` — the collection is
   the value keyed by the command name itself); it's captured there and
   correlated back to the matching `Succeeded`/`Failed` event by
   `RequestID` in an internal mutex-guarded map.
3. **Go runtime / process metrics** — `go_goroutines`,
   `go_memstats_heap_inuse_bytes`, `process_cpu_seconds_total`, etc. —
   registered automatically by `promauto` alongside the two metric families
   above; no separate wiring needed.

`metrics.Handler()` (`gin.WrapH(promhttp.Handler())`) is what `GET /metrics`
actually serves — registered on the `root` group next to `/health` in
`apps/api/app/app.go`, so it goes through the same middleware stack
(RequestID/Recovery/Logger/gzip/CORS/RateLimiter/etc.) as every other route,
same as `/health` — with one deliberate exception: `/metrics` is passed to
`gzip.WithExcludedPaths` so the global gzip middleware skips it.
`promhttp.Handler()` already gzips its own response whenever
`Accept-Encoding: gzip` is present (which Prometheus's scraper always sends),
so without the exclusion the gzip middleware would compress that
already-gzipped body a second time; Prometheus strips only one
`Content-Encoding: gzip` layer, so it would try to parse the inner layer's
raw gzip bytes as plaintext and fail every scrape (verified live against
Prometheus v2.55.1: `"expected a valid start token, got \"\x1f\""` — 0x1f is
the gzip magic byte).

- **Prometheus + Grafana stack:** `make docker-monitoring` starts Prometheus
  (`monitoring/prometheus.yml` — scrapes `app:3000/metrics` every 15s) and
  Grafana (`monitoring/grafana/provisioning/` — auto-provisions the
  Prometheus datasource and the **Bread API Overview** dashboard from
  `monitoring/grafana/dashboards/bread-api-overview.json`, no manual
  "Add data source"/"Import dashboard" click needed) on the `monitoring`
  Compose profile, independent of and combinable with every other profile
  (`worker`/`rabbitmq`/`kafka`/`mqtt`) — same pattern as `docker-mqtt`.
  Prometheus UI → `http://localhost:9090`; Grafana → `http://localhost:3001`
  (`admin`/`admin` — Grafana's own default port 3000 is already the API's,
  so the host mapping is `3001:3000`).
- **No config required:** unlike Firebase/MQTT/etc., there's no
  `*_ENABLED`/`*_URL` env var gating this — `/metrics` is always exposed
  (matching `/health`'s always-on convention), so `make docker-monitoring`
  works out of the box against a running `app` container with zero `.env`
  changes.
- **Dashboard panels:** HTTP request rate by status, HTTP p95 latency by
  route, HTTP error rate (4xx/5xx), MongoDB operation rate by operation,
  MongoDB p95 latency by operation, MongoDB error rate by
  operation+collection, Go goroutines, Go heap in use, process CPU — see
  `monitoring/grafana/dashboards/bread-api-overview.json` for the exact
  PromQL behind each panel.
- Not unit-tested via real Prometheus scrape/Grafana render (out of scope
  for unit tests, same reasoning as every other real-infra boundary in this
  project) — `pkg/metrics/metrics_test.go` exercises `GinMiddleware` via
  `httptest`/`gin.CreateTestContext`-style requests and
  `MongoCommandMonitor`'s `Started`/`Succeeded`/`Failed` callbacks directly
  with hand-built `event.Command*Event` structs, asserting via
  `prometheus/client_golang/prometheus/testutil`.

---

## Multi-language (i18n)

Locale files: `locales/en.json` (default), `locales/id.json`

Key structure:
```
http.*          ← HTTP status messages
auth.*          ← Authentication messages
user.*          ← User CRUD messages
role.*          ← Role messages
passkey.*       ← WebAuthn messages
mobile.*        ← OTP / mobile verification
appVersion.*    ← App version check messages
notification.*  ← Notification messages
realtime.*      ← WebSocket/SSE/pub-sub messages
iot.*           ← MQTT device-telemetry demo messages
analytics.*     ← Analytics endpoint messages
apiKey.*        ← API key messages
tenant.*        ← Multi-tenant messages
featureFlag.*   ← Feature flag messages
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
- Prefer adding keys to BOTH `en.json` and `id.json`. `apiKey.*`, `passkey.*`,
  `tenant.*`, `role.*`, and `featureFlag.*` are the current exception —
  they're English-only today and intentionally rely on the fallback below;
  don't feel obligated to add Indonesian translations there unless you're
  doing a real localization pass. `auth.*`, `user.*`, `mobile.*`,
  `appVersion.*`, `notification.*`, `realtime.*`, `iot.*`, `analytics.*`,
  `http.*`, `health.*`, and `validation.*` are fully bilingual — keep new
  keys in those namespaces bilingual too.
- `validation.*` covers both the per-field validator-tag messages
  (`validation.required`, `validation.min`, …) AND the two response-envelope
  messages `shared/validate.BindJSON`/`BindQuery` use directly:
  `validation.failed` (422, struct validation failed) and
  `validation.invalidBody`/`validation.invalidQuery` (400, malformed
  JSON/query — only the prefix is translated, the appended Go parse error
  stays raw since it isn't meaningfully translatable).
- Use `{name}`, `{field}` etc. for variable interpolation (NOT Go template syntax)
- Missing keys in non-default languages fall back to `en` automatically
- Every handler response should go through `response.*I18n` with a locale
  key — see "Error handling in handlers" and "Response envelope" above.
  Don't hardcode a client-facing message string in a handler.
- To manually compare how a response looks in two languages, use the web
  test client's **i18n Compare** page (`web/src/pages/I18nComparePage.tsx`,
  route `/i18n-compare`) — it fires the same request twice with different
  `x-custom-lang` headers and shows both responses side by side.

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

Run `make swagger` after adding/changing any annotations. Under `make dev`
this happens automatically on every hot reload — `.air.toml`'s build step is
`scripts/dev/air-build-api.sh`, which runs `swag init` before every `go
build`, so annotation changes show up in Swagger UI on the next save without
a manual `make swagger`. Only `make dev` does this; `make run`/`make build`
and the worker's `make dev-worker*` targets don't touch Swagger.

---

## Web Test Client

`web/` is a React + TypeScript SPA (Vite, Tailwind, React Router) that
exercises every HTTP-exposed endpoint of `apps/api` — auth, 2FA, passkeys
(real WebAuthn ceremonies via `@simplewebauthn/browser`), mobile OTP,
notifications, admin users/roles/app-versions/analytics/notifications,
realtime WebSocket/SSE, IoT/MQTT, health — plus a generic raw-request API
console, an **i18n Compare** page (`/i18n-compare`) that fires one request
twice with different `x-custom-lang` headers to compare translated
responses side by side, and a client-side activity log for anything not
covered by a dedicated page. It talks to the API directly over `fetch`
(CORS already allows `*`) or native `WebSocket`/`EventSource`; no backend
code changes were needed to support any of it.

**Realtime** (`/realtime`, `web/src/pages/RealtimePage.tsx`) — connect/
disconnect panels for both `GET /v1/me/ws` and `GET /v1/me/events`, each
with a live event log; the WebSocket panel can also join/leave an arbitrary
topic via a subscribe/unsubscribe control frame. Below that, an admin
**Publish to topic** form (`POST /v1/admin/realtime/publish`) for the
generic pub/sub path, and a **stats** panel
(`GET /v1/admin/realtime/stats`). The page shows your own private channel
name (`user:<your-id>`) — every notification sent to you via Admin →
Notifications shows up here live in addition to being persisted.

**Admin → IoT** (`/admin/iot`, `web/src/pages/IotPage.tsx`) — **Simulate**
publishes a reading to MQTT as if a real device sent it, **Send command**
publishes in the other direction, and a paginated table shows persisted
readings from `GET /v1/admin/iot/telemetry`. Open the Realtime page in
another tab (WS or SSE, topic `iot:telemetry`) before clicking Simulate to
watch the full MQTT → subscriber → Mongo → realtime-push chain fire live.
Requires `MQTT_BROKER_URL` to be configured (`make docker-mqtt`) or both
forms return a 503.

**Admin → Notifications** (`/admin/notifications`,
`web/src/pages/AdminNotificationsPage.tsx`) is the page for manually testing
queue routing: a **Test email** form (`test-email`, synchronous, surfaces the
raw transport error), a **Send to one user** form (`admin/notifications/send`,
always synchronous), and a **Broadcast** form (`admin/notifications/broadcast`)
with a user-checkbox picker — broadcasting to the `email` channel is the one
real trigger for `EnqueuePromotionalEmail`/`QUEUE_PROMOTIONAL_DRIVER`, so it's
the fastest way to see the transactional/promotional split in action: create
a user from **Admin → Users** first (fires the transactional welcome email)
and watch which worker log picks it up, then broadcast to the `email`
channel and watch a *different* worker (whichever runs
`QUEUE_PROMOTIONAL_DRIVER`) pick that one up instead.

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
pkg/realtime/hub_test.go                    # fake in-memory Client, no network
modules/realtime/service/realtime.service_test.go
modules/iot/service/iot.service_test.go     # fake MQTT Publisher, no real broker
pkg/metrics/metrics_test.go                 # httptest requests + hand-built event.Command*Event structs
```

Use fake/stub implementations, never real MongoDB/Redis in unit tests. The
MQTT-subscribe side (`IoTService.Start`/`handleTelemetry`) and any
`NotificationService`/`Broadcast` path that touches `s.prefsCol`/`s.deviceCol`
aren't unit-tested for the same reason — they need a real `*mongo.Collection`
and belong in `tests/e2e` instead (see the live-verification notes in each
service's test file for what's covered vs. not).

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
APP_URL=http://localhost:5173  # frontend base URL — used to build links in outbound emails (dashboard/docs/verify/reset). Defaults to the web/ test client.
MONGO_URI=mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0
REDIS_HOST=redis
JWT_ACCESS_PRIVATE_KEY_PATH=./keys/access_private.pem
JWT_ACCESS_PUBLIC_KEY_PATH=./keys/access_public.pem
JWT_REFRESH_PRIVATE_KEY_PATH=./keys/refresh_private.pem
JWT_REFRESH_PUBLIC_KEY_PATH=./keys/refresh_public.pem

# Optional integrations
MAIL_DRIVER=ses                # "ses" (default, needs AWS_ACCESS_KEY_ID/AWS_SES_*) or "smtp" (needs SMTP_*)
SMTP_HOST=smtp.gmail.com       # for personal Gmail use smtp.gmail.com, NOT smtp.google.com (Workspace relay — rejects personal accounts)
AWS_ACCESS_KEY_ID=             # enables SES email (if MAIL_DRIVER=ses) + S3
TWILIO_ACCOUNT_SID=            # enables SMS + WhatsApp OTP
FIREBASE_CREDENTIALS_FILE=     # enables FCM push notifications
GITHUB_CLIENT_ID=              # enables GitHub SSO
APPLE_CLIENT_ID=               # enables Apple Sign In
SENTRY_DSN=                    # enables error tracking
MULTI_TENANT_ENABLED=false     # enable multi-tenant mode

# Background job queue backend — MUST match the worker process you run
QUEUE_DRIVER=redis             # "redis" (default, apps/worker) | "rabbitmq" (apps/worker-rabbitmq) | "kafka" (apps/worker-kafka)
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/   # used when QUEUE_DRIVER=rabbitmq
KAFKA_BROKERS=kafka:9092                          # used when QUEUE_DRIVER=kafka

# MQTT (modules/iot demo) — unrelated to QUEUE_DRIVER above. Blank (default)
# disables modules/iot entirely (every call returns 503).
MQTT_BROKER_URL=               # e.g. tcp://mosquitto:1883 — set to use `make docker-mqtt`

# Monitoring — no env var needed. GET /metrics is always exposed (pkg/metrics,
# same tier as /health); `make docker-monitoring` starts Prometheus + Grafana
# to scrape/visualize it.
```

In Docker, `MONGO_URI` and `REDIS_HOST` are always overridden by
`docker-compose.yml` `environment:` block to use Docker service names.

**Custom env file for local scripts:**
```bash
# make seed-local and make migrate-indexes-local read .env by default.
# Override with:
make seed-local ENV_FILE=.env.staging
# Or directly:
BREAD_CONFIG_FILE=/path/to/.env go run ./scripts/seed
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
| Prometheus | `localhost:9090` | Scrapes the API's `/metrics` (target: `devcontainer:3000`, not `app:3000` — see below) |
| Grafana | `localhost:3001` (`admin`/`admin`) | Pre-provisioned "Bread API Overview" dashboard |
| (API) | `localhost:3000` | Started manually via `make dev` |
| (Web test client) | `localhost:5173` | Started manually via `make web-dev` |

Unlike the root `docker-compose.yml`'s opt-in `monitoring` Compose profile
(`make docker-monitoring`), Prometheus and Grafana run unconditionally in
the dev container — same convenience rationale as RabbitMQ/Kafka always
running here too. `.devcontainer/docker-compose.yml`'s `prometheus` service
mounts `monitoring/prometheus.devcontainer.yml`, a separate scrape config
from the root's `monitoring/prometheus.yml` — the dev container runs the API
via `make dev` directly inside the `devcontainer` service (no separate `app`
container exists there), so the scrape target is `devcontainer:3000`
instead of `app:3000`. Both configs keep `job_name: bread-api` so the same
provisioned Grafana dashboard (which filters some panels by `job="bread-api"`)
works unmodified in both environments.

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
make docker-worker      # same + Redis/Asynq background worker (profile: worker)
make docker-rabbitmq    # same + RabbitMQ broker + RabbitMQ worker (profile: rabbitmq)
make docker-kafka       # same + Kafka broker + Kafka worker (profile: kafka)
make docker-mqtt        # same + Mosquitto MQTT broker (profile: mqtt, modules/iot demo)
make docker-monitoring  # same + Prometheus + Grafana (profile: monitoring)
make docker-down        # stop all (all five profiles above)
make docker-rebuild     # rebuild + restart app container only
make docker-clean       # remove containers AND volumes (destructive)
make seed               # seed DB (runs inside Docker network, requires docker-up)
make migrate-indexes    # create MongoDB indexes (inside Docker network)
make seed-local         # seed against localhost:27017 (for local dev without Docker)
```

`docker-worker`, `docker-rabbitmq`, and `docker-kafka` are mutually exclusive
choices of **one** queue backend, not additive — each is a separate Compose
profile (`worker` / `rabbitmq` / `kafka`) that starts its own broker (where
applicable) and matching worker container. Set `QUEUE_DRIVER` in `.env` to
match whichever one you start. See "Background Job Queue Backends" above.
`docker-mqtt` (profile `mqtt`) and `docker-monitoring` (profile `monitoring`)
are each independent of all three — `docker-mqtt` is the `modules/iot`
demo's own broker, not a `QUEUE_DRIVER` choice; `docker-monitoring` is
Prometheus + Grafana scraping the always-on `/metrics` endpoint. Both combine
freely with any of the above and with each other.

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
| Returning raw Go error strings to the client | Wrap in `errors.AppError` and call `response.HandleAppError(c, err)` |
| Hardcoding a client-facing message in `response.OK`/`Created`/`Error`/etc. | Use the `*I18n` variant with a `locales/*.json` key (`response.OKI18n`, `response.ErrorI18n`, ...) — see "Response envelope" |
| Building a new `errors.AppError` sentinel with `errors.New(...)` | Use `errors.NewI18n(status, code, key, message)` so the client-facing text is translated via `x-custom-lang`, not just the internal `Message` |
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
| Gmail SMTP fails to connect / auth with `SMTP_HOST=smtp.google.com` | `smtp.google.com` is the Workspace relay; personal `@gmail.com` accounts must use `SMTP_HOST=smtp.gmail.com` (587 STARTTLS or 465 SSL) with a 16-char **App Password** (2FA required), not the account password |
| Welcome email (or other transactional email) never arrives even though config is correct | Request-triggered emails are enqueued to the asynq queue and delivered by the **worker** — run `make docker-worker`/`make run-worker`/`make dev-worker`; `make dev` alone only enqueues. See "Async delivery via the worker queue" |
| Enqueuing email from a service by importing `asynq` / task-type strings | Depend on the `EmailQueue` interface (`EnqueueEmail(to,subject,html,text)`) — `worker.Client` implements it; render via `LocalizedMailer` then enqueue |
| Adding a request-triggered email as a synchronous `localMailer.Send*` call | Only diagnostics (e.g. `test-email`) send synchronously — enqueue everything else via `jobQueue.EnqueueEmail` so a slow/failed SMTP send doesn't block or fail the HTTP request |
| Jobs enqueued but never processed, no error anywhere | `QUEUE_DRIVER` doesn't match the worker process actually running — e.g. `QUEUE_DRIVER=rabbitmq` in `.env` but `make docker-worker` (Redis) is what's running. Start the worker whose profile matches `QUEUE_DRIVER` (`docker-worker`/`docker-rabbitmq`/`docker-kafka`) |
| Set `QUEUE_TRANSACTIONAL_DRIVER`/`QUEUE_PROMOTIONAL_DRIVER` but promotional/transactional jobs still silent | Same rule as `QUEUE_DRIVER`, per-route — the worker for THAT driver must be running too. `make docker-queues` starts all three at once instead of picking one via `docker-worker`/`docker-rabbitmq`/`docker-kafka` |
| Importing `asynq`, `amqp091-go`, or `kafka-go` directly in a `modules/*/service` file | Depend on the `EmailQueue` interface only — the concrete `Publisher` (`worker.Client` / `rabbitmq.Publisher` / `kafka.Publisher`) is chosen once in `apps/api/app/app.go`'s `buildJobQueue` |
| `go: missing go.sum entry` for `amqp091-go` or `kafka-go` after a fresh clone | Run `make deps` (go get both client libs + `go mod tidy`) |
| `POST /v1/admin/notifications/send` or `/broadcast` with `channel: "email"` returns 400 "Recipient email address is required" | `NotificationService.sendEmail` has no `UserRepo` to look up the recipient's real address yet — include `"data": {"email": "..."}` in the request body. This used to surface as an opaque 500 (`errors.ErrEmailNotAvailable`/`ErrMailerNotConfigured` now map it to a proper 4xx via `response.HandleAppError`) |
| `make dev` feels slow on every save, even for unrelated changes | Expected — `scripts/dev/air-build-api.sh` runs `swag init` (a few seconds) before every rebuild by design, so Swagger docs never go stale. Email templates only rebuild when `email-templates/src` actually changed |
| Edited an email `.tsx` under `make dev` but the rendered HTML didn't change | Check `.air/build.log` for an `npm run build` failure; also confirm Node.js is on `PATH` — the script skips the email rebuild (with a warning) if `node` isn't found, but still regenerates Swagger and builds the Go binary |
| Browser `WebSocket`/`EventSource` to `/v1/me/ws` or `/v1/me/events` gets 401 | Neither API can set an `Authorization` header — pass the access token as `?token=` instead; that's why those two routes use `middleware.AuthJWTAccessWS`, not the regular `authMw` |
| `POST /v1/admin/iot/devices/:id/simulate` or `/command` returns 503 | `MQTT_BROKER_URL` is unset or the broker dial failed at startup (check the API log for "MQTT connect failed") — set it and run `make docker-mqtt`, then restart the API |
| Simulated telemetry never shows up on the Realtime page | The WS/SSE client must be subscribed to topic `iot:telemetry` specifically — the private per-user channel doesn't receive it. Also confirm `MQTT_BROKER_URL` is actually configured, not just that `/simulate` returned 200 (publish succeeding doesn't guarantee a subscriber is running if `IoTService.Start()` failed — check the log) |
| Adding a `pkg/mqtt`/`pkg/realtime` capability directly in a `modules/*/service` file | Depend on a small local interface (`Publisher`, `RealtimePublisher`, ...) satisfied structurally by the concrete type — same pattern as `EmailQueue` — never import `paho.mqtt.golang` or `gorilla/websocket` outside `pkg/mqtt`/`pkg/realtime` and the `modules/realtime` handler |
| `GET /metrics` returns an empty/near-empty scrape | Metrics are recorded on demand — `http_requests_total`/`mongodb_operations_total` label combinations only appear after at least one matching request/command has happened; hit a route (or run `make seed`) first, then re-scrape. Go runtime metrics (`go_goroutines`, etc.) are always present regardless |
| Calling `database.NewMongoDB` instead of `NewMongoDBWithMonitor` and expecting MongoDB metrics/logs | Only `apps/api/main.go` passes `metrics.MongoCommandMonitor(log)` in — `scripts/seed`, `scripts/migrate`, and the three worker `main.go`s still call plain `NewMongoDB` (nil monitor), by design, since one-shot scripts and workers aren't scraped |
| Grafana shows "No data" right after `make docker-monitoring` | Prometheus's first scrape can take up to `scrape_interval` (15s in `monitoring/prometheus.yml`) to land, and every panel needs at least one matching HTTP request or MongoDB command to have happened on the `app` container in that window — hit a few routes, then wait/refresh |
| Grafana UI won't load at `http://localhost:3000` | That's the API's port — Grafana is mapped to `3001:3000` in `docker-compose.yml` specifically to avoid the collision; use `http://localhost:3001` |
| Prometheus scrape of `/metrics` fails with `"expected a valid start token, got \"\x1f\""` | Would happen if `/metrics` were ever removed from `gzip.WithExcludedPaths` in `apps/api/app/app.go` — `promhttp.Handler()` already gzips its own response when `Accept-Encoding: gzip` is present (Prometheus always sends it), so the global gzip middleware would double-compress it; Prometheus only strips one `Content-Encoding: gzip` layer and chokes on the inner layer's raw gzip magic byte. Keep `/metrics` excluded |

---

## Key Files to Know

| File | Purpose |
|---|---|
| `apps/api/app/app.go` | All dependency wiring — edit when adding a module |
| `shared/config/config.go` | Add new env vars (struct + Load() + SetDefault + viper default) |
| `shared/validate/validate.go` | Single validation entry point for all handlers |
| `shared/response/response.go` | Standard response envelope helpers |
| `shared/response/response_i18n.go` | `*I18n` response helpers + `HandleAppError` — the single error→response translation point |
| `shared/errors/errors.go` | Domain error sentinels (add new ones here, via `errors.NewI18n`) |
| `modules/role/repository/role.repository.go` | Role lookup — `FindByID` used during JWT issuance, `FindAll` for the roles dropdown |
| `modules/role/handler/role.handler.go` | `GET /v1/roles` (admin) — lists roles to populate the create-user role dropdown in `web/` |
| `pkg/worker/worker.go` | Asynq (Redis) client/server + task types; `Client.EnqueueEmail` is the transactional-email entry point |
| `pkg/worker/tasks/tasks.go` | Task handlers run by the Redis/Asynq worker (`email:send`, session cleanup, …) |
| `pkg/queue/queue.go` | Backend-agnostic `Publisher`/`Consumer` contract shared by RabbitMQ + Kafka; `EmailQueue` callers depend on this shape |
| `pkg/queue/router/router.go` | Routes task types to different `Publisher`s (`QUEUE_TRANSACTIONAL_DRIVER`/`QUEUE_PROMOTIONAL_DRIVER`); itself satisfies `queue.Publisher` |
| `pkg/queue/rabbitmq/rabbitmq.go` | RabbitMQ `Publisher`/`Consumer` impl (AMQP exchange + durable queue) |
| `pkg/queue/kafka/kafka.go` | Kafka `Publisher`/`Consumer` impl (topic + consumer group, `segmentio/kafka-go`) |
| `pkg/queue/tasks/tasks.go` | Task handlers shared by `apps/worker-rabbitmq` and `apps/worker-kafka` |
| `apps/api/app/app.go` (`buildJobQueue`) | Picks the enqueue-side `Publisher` from `QUEUE_DRIVER` — must match the worker process actually running |
| `pkg/realtime/hub.go` | `Hub` — the WebSocket/SSE fan-out core (`Subscribe`/`Unsubscribe`/`Publish`/`Stats`), zero domain knowledge |
| `modules/realtime/service/realtime.service.go` | `UserTopic(id)` naming convention + the `realtime.Service` contract implementation other modules call |
| `modules/realtime/handler/realtime.handler.go` | `GET /v1/me/ws` (gorilla/websocket upgrade), `GET /v1/me/events` (SSE via `c.Stream`), admin publish/stats |
| `shared/middleware/auth.go` (`AuthJWTAccessWS`) | JWT middleware variant accepting `?token=` — used only by the two realtime routes above |
| `pkg/mqtt/client.go` | Thin `paho.mqtt.golang` wrapper — `Connect`/`Publish`/`Subscribe`/`Close` |
| `modules/iot/service/iot.service.go` | `SimulateTelemetry`/`SendCommand` (publish side) + `Start`/`handleTelemetry` (subscribe side, persists + forwards to `modules/realtime`) |
| `pkg/metrics/metrics.go` | `GinMiddleware()` (HTTP metrics), `MongoCommandMonitor(log)` (MongoDB metrics + zap logs), `Handler()` (`/metrics` scrape endpoint) |
| `monitoring/prometheus.yml` | Prometheus scrape config — targets `app:3000/metrics` |
| `monitoring/grafana/dashboards/bread-api-overview.json` | The pre-provisioned Grafana dashboard — edit panels/PromQL here |
| `modules/activity/service/activity.service.go` | Action constants + log helpers |
| `modules/<name>/contract.go` | Public interface — the only safe cross-module import |
| `pkg/dbid/dbid.go` | ID strategy helpers (uuid / objectid) — see `docs/id-migration.md` |
| `pkg/email/localized_mailer.go` | Render*/Send* email methods |
| `pkg/email/factory.go` | `NewMailerFromConfig` — picks SES vs SMTP via `MAIL_DRIVER` |
| `locales/en.json` + `locales/id.json` | All i18n strings including email tokens |
| `scripts/migrate/main.go` | All MongoDB index definitions |
| `scripts/seed/` | Default data — one file per module (`role.go`, `user.go`, `featureflag.go`, `appversion.go`), wired together by `main.go` |
| `docs/id-migration.md` | How to switch between UUID and ObjectID |
| `docs/email-i18n.md` | Email template architecture documentation |
| `docs/feature-parity.md` | Full NestJS ↔ Go feature mapping |
| `web/README.md` | Web test client — running it, WebAuthn origin setup, OAuth caveats |
| `web/src/lib/apiClient.ts` | Test client's fetch wrapper — auth headers, refresh-on-401, request logging |
| `web/src/api/*.ts` | Typed wrappers per module, mirroring each `dto` package 1:1 |
| `web/src/pages/I18nComparePage.tsx` | `/i18n-compare` — fires one request twice with different `x-custom-lang` headers to compare translated responses side by side |
