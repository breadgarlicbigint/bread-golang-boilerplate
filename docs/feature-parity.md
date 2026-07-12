# Feature Parity Report

Mapping of every feature in `andrechristikan/ack-nestjs-boilerplate` to this Go boilerplate.

---

## 🎯 Architecture Highlights

| Feature | NestJS | Go | File |
|---|---|---|---|
| Repository Pattern | ✅ | ✅ | `internal/*/repository/` |
| SOLID Principles | ✅ | ✅ | Interfaces throughout |
| Modular Structure | ✅ | ✅ | `internal/<module>/{entity,dto,repository,service,handler}` |
| 12-Factor App | ✅ | ✅ | Viper + env vars, `config/config.go` |
| Production Ready | ✅ | ✅ | All security layers present |

---

## 🔐 Authentication & Security

| Feature | NestJS | Go | File |
|---|---|---|---|
| JWT ES256 Access Token | ✅ | ✅ | `pkg/jwt/jwt.go` |
| JWT ES512 Refresh Token | ✅ | ✅ | `pkg/jwt/jwt.go` |
| Token Rotation | ✅ | ✅ | `auth/service/auth.service.go` → `Refresh()` |
| Stateful Sessions (Redis) | ✅ | ✅ | `session:` key prefix in Redis |
| Token Revocation | ✅ | ✅ | `RevokeSession()`, `LogoutAll()` |
| Google OAuth | ✅ | ✅ | `auth/service/google.go` (original) |
| Apple Sign In | ✅ | ✅ | `auth/service/apple.go`, `auth/handler/apple.handler.go` |
| GitHub SSO | ❌ | ✅ | `auth/service/github.go`, `auth/handler/github.handler.go` |
| TOTP 2FA | ✅ | ✅ | `auth/service/auth.service.go` → `Enable2FA()` |
| 2FA Backup Codes | ✅ | ✅ | 8 backup codes generated on enable |
| Passkey / WebAuthn | ❌ | ✅ | `passkey/` module, `go-webauthn/webauthn` |
| Biometric Login | ❌ | ✅ | Passkey with `attachment=platform` |
| RBAC | ✅ | ✅ | `role/entity`, `middleware/auth.go` → `RoleProtected()` |
| Policies / Permissions | ✅ | ✅ | `role/entity/role.entity.go` → `Permission` constants |
| API Key Protection | ✅ | ✅ | `apikey/service`, `middleware/apikey.go` |
| Rate Limiting | ✅ | ✅ | `middleware/ratelimit.go` (Redis sliding window) |
| Security Headers (Helmet) | ✅ | ✅ | `middleware/security.go` → `SecurityHeaders()` |
| Brute-force lockout | ✅ | ✅ | `user/service` → `HandleFailedLogin()` |

---

## 📊 Database & Storage

| Feature | NestJS | Go | File |
|---|---|---|---|
| MongoDB (replica set) | ✅ | ✅ | `database/mongo.go` |
| Transaction support | ✅ | ✅ | `database.Transactor` |
| Redis Caching | ✅ | ✅ | `database/redis.go`, analytics cache |
| AWS S3 + presigned URLs | ✅ | ✅ | `pkg/storage/s3.go` |
| Repository Pattern | ✅ | ✅ | All modules have `repository/` layer |
| Soft Deletes | ✅ | ✅ | `deletedAt` on users |
| TTL Indexes (auto-expire) | ✅ | ✅ | Activity logs & notifications: 90 days |
| Prisma ORM | ✅ | N/A | Replaced by mongo-driver + Repository Pattern |

---

## ⚡ Performance & Optimization

| Feature | NestJS | Go | File |
|---|---|---|---|
| Background Jobs (BullMQ) | ✅ | ✅ | `pkg/worker/worker.go` (Asynq) |
| Response Compression | ✅ | ✅ | `gin-contrib/gzip` in `app.go` |
| Pagination (offset) | ✅ | ✅ | `common/pagination/pagination.go` |
| Cursor Pagination | ✅ | ✅ | `pagination.Query.Cursor` field |
| Feature Flags | ✅ | ✅ | `featureflag/service`, Redis cache |
| SWC Compiler | ✅ | N/A | Go compiles natively (~faster) |

---

## 🛠 Development Experience

| Feature | NestJS | Go | File |
|---|---|---|---|
| Swagger / OpenAPI 3 | ✅ | ✅ | swaggo/swag annotations, `make swagger` |
| API Versioning (v1) | ✅ | ✅ | `/v1` group in `app.go` |
| Request Validation | ✅ | ✅ | go-playground/validator in handlers |
| Standardised Errors | ✅ | ✅ | `common/errors/errors.go`, `common/response/` |
| i18n (x-custom-lang) | ✅ | ✅ | `pkg/i18n/i18n.go`, `locales/en.json`, `locales/id.json` |
| Hot Reload | ✅ | ✅ | air + `.air.toml` |
| Code Quality (ESLint equiv.) | ✅ | ✅ | golangci-lint + `.golangci.yml` |
| Database Seeding | ✅ | ✅ | `scripts/seed/` (one file per module) |
| TypeScript | ✅ | N/A | Go is statically typed |

---

## 📡 Integrations & Monitoring

| Feature | NestJS | Go | File |
|---|---|---|---|
| Sentry | ✅ | ✅ | `app.go` → `sentry.Init()` |
| AWS SES | ✅ | ✅ | `pkg/email/ses.go` |
| AWS SES Email Templates | ✅ | ✅ | `pkg/email/ses.go` → `TemplateMail()` |
| Activity Logging | ✅ | ✅ | `activity/service/` |
| Bidirectional Activity Logging | ❌ | ✅ | `DirectionInbound` / `DirectionOutbound` |
| Health Checks | ✅ | ✅ | `/health`, `/health/live`, `/health/ready` |
| Multi-language (i18n) | ✅ | ✅ | `pkg/i18n/`, `x-custom-lang` header |

---

## 🔔 Notifications

| Feature | NestJS | Go | File |
|---|---|---|---|
| Email Notifications | ✅ | ✅ | `notification/service` → `sendEmail()` |
| Push Notifications (FCM) | ✅ | ✅ | `notification/service/fcm.go` |
| Multicast Push | ✅ | ✅ | `FCMSender.SendMulticast()` + stale token cleanup |
| In-App Notifications | ✅ | ✅ | `ChannelInApp` persisted in MongoDB |
| Silent Notifications | ✅ | ✅ | `ChannelSilent` → data-only FCM message |
| Per-Channel Preferences | ✅ | ✅ | `NotificationPreferences.Channels` map |
| Per-Type Preferences | ✅ | ✅ | `NotificationPreferences.Types` nested map |
| Queue-Based Delivery | ✅ | ✅ | `pkg/worker/tasks/notification_tasks.go` (Asynq) |
| Stale Token Cleanup | ✅ | ✅ | `SendMulticast()` auto-removes invalid tokens |
| Device Token Registration | ✅ | ✅ | `POST /v1/me/notifications/devices` |
| Notification TTL (90 days) | ✅ | ✅ | MongoDB TTL index on `createdAt` |

---

## ➕ Added Features (beyond original)

| Feature | File |
|---|---|
| GitHub SSO | `auth/service/github.go` |
| Passkey / FIDO2 | `passkey/` module |
| Biometric Login | Passkey with `platform` attachment |
| Bidirectional Activity Logging | `activity/service/` |
| Multi-Tenant Architecture | `tenant/` module, `tenant/middleware/` |
| Mobile Number Verification (SMS + WhatsApp) | `mobile/` module, `pkg/sms/` |
| App Versioning System | `appversion/` module |
| Analytics Dashboard | `analytics/` module (13 endpoints) |
| Fraud Scoring | `analytics/service` → `FraudSignalSummary()` |
| Anomaly Detection | Credential stuffing, device proliferation |

---

## 🧪 Testing

| Feature | NestJS | Go | File |
|---|---|---|---|
| Unit Tests | ✅ | ✅ | `*_test.go` in every service package |
| Integration Tests | ✅ | ✅ | `tests/integration/auth_test.go` |
| E2E Tests | ✅ | ✅ | `tests/e2e/e2e_test.go` |
| Test Coverage Report | ✅ | ✅ | `make test-coverage` → `coverage.html` |

---

## 🐳 Infrastructure

| Feature | NestJS | Go | File |
|---|---|---|---|
| Docker (multi-stage) | ✅ | ✅ | `Dockerfile` |
| Docker Compose | ✅ | ✅ | `docker-compose.yml` |
| MongoDB Replica Set | ✅ | ✅ | 3-node rs0 in compose |
| Redis | ✅ | ✅ | Redis 8 in compose |
| Health Check in Dockerfile | ✅ | ✅ | `HEALTHCHECK` on `/health/live` |

---

## Complete Route Table

```
── Public ──────────────────────────────────────────────────────────────────────
GET    /health
GET    /health/live
GET    /health/ready
GET    /docs/*any                           (non-production only)

POST   /v1/auth/login
POST   /v1/auth/register
POST   /v1/auth/refresh
GET    /v1/auth/google                      → redirect to Google
GET    /v1/auth/google/callback
GET    /v1/auth/github                      → redirect to GitHub
GET    /v1/auth/github/callback
POST   /v1/auth/apple/callback              (iOS/Android send id_token directly)
POST   /v1/auth/passkey/login/begin
POST   /v1/auth/passkey/login/finish
POST   /v1/auth/passkey/identified/begin
POST   /v1/auth/passkey/identified/finish
GET    /v1/app-version/check

── Bearer (authenticated user) ─────────────────────────────────────────────────
DELETE /v1/auth/logout
DELETE /v1/auth/logout-all
POST   /v1/auth/2fa/enable
POST   /v1/auth/2fa/verify

GET    /v1/me
PATCH  /v1/me
PATCH  /v1/me/password

POST   /v1/me/passkeys/register/begin
POST   /v1/me/passkeys/register/finish
GET    /v1/me/passkeys
DELETE /v1/me/passkeys/:id

POST   /v1/me/mobiles/send-otp
POST   /v1/me/mobiles/verify
GET    /v1/me/mobiles
PATCH  /v1/me/mobiles/:e164/primary
DELETE /v1/me/mobiles/:e164

GET    /v1/me/notifications
GET    /v1/me/notifications/unread-count
PATCH  /v1/me/notifications/:id/read
PATCH  /v1/me/notifications/read-all
GET    /v1/me/notifications/preferences
PATCH  /v1/me/notifications/preferences
POST   /v1/me/notifications/devices
DELETE /v1/me/notifications/devices/:token

── Admin (Bearer + admin role) ─────────────────────────────────────────────────
POST   /v1/users
GET    /v1/users
GET    /v1/users/:id
PATCH  /v1/users/:id
DELETE /v1/users/:id
POST   /v1/users/:id/block
POST   /v1/users/:id/unblock

GET    /v1/admin/app-versions
PUT    /v1/admin/app-versions/:platform

POST   /v1/admin/notifications/send
POST   /v1/admin/notifications/broadcast

GET    /v1/admin/analytics/users/registrations
GET    /v1/admin/analytics/users/churn
GET    /v1/admin/analytics/users/signup-methods
GET    /v1/admin/analytics/users/blocked-trend
GET    /v1/admin/analytics/auth/login-frequency
GET    /v1/admin/analytics/auth/login-methods
GET    /v1/admin/analytics/auth/lockout
GET    /v1/admin/analytics/passkeys/adoption
GET    /v1/admin/analytics/mobile/verification
GET    /v1/admin/analytics/anomalies/credential-stuffing
GET    /v1/admin/analytics/anomalies/device-proliferation
GET    /v1/admin/analytics/fraud/signals
```
