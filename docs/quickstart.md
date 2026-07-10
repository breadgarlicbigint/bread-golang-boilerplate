# Quick Start

## Prerequisites

| Tool | Min version | Install |
|------|-------------|---------|
| Go | 1.23 | https://go.dev/dl |
| Node.js | 20+ | https://nodejs.org — needed for `make build-emails` only |
| Docker | 24 | https://docs.docker.com/get-docker |
| Docker Compose | v2 | bundled with Docker Desktop |
| OpenSSL | any | `brew install openssl` / `apt install openssl` |

---

## One-command setup (recommended)

```bash
git clone https://github.com/breadgarlicbigint/bread-golang-boilerplate
cd bread-golang-boilerplate

# Runs: cp .env.example, generate-keys, go mod tidy, swagger
make setup

# Review and fill in .env (AWS, Twilio, Firebase are optional)
nano .env

# Start MongoDB replica set + Redis + API
make docker-up

# Populate database (roles, users, feature flags, app versions)
# NOTE: 'make seed' runs INSIDE the Docker network so mongo1/mongo2/mongo3
# hostnames resolve. It uses 'docker run' with the golang:1.23-alpine image.
make seed

# Smoke-test
curl http://localhost:3000/health
curl -X POST http://localhost:3000/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"Admin@1234"}'
```

---

## Why `make mod-sync` / `go mod tidy` matters

Go requires `go.sum` to be present before `go mod download` can verify module
integrity. The repo ships an **empty** `go.sum`; the first thing you must do is:

```bash
go mod tidy        # or: make mod-sync
```

This:
- Downloads all declared dependencies
- Hashes them and writes the results to `go.sum`
- Removes any stale entries from `go.mod`

`make setup` does this automatically. If you skip it and run `docker compose up`
directly, the Docker builder will fall back to `go mod tidy` inside the container
(safe but slower and requires internet access at build time).

---

## Common Docker Issues

### `go.sum: file does not exist`

**Cause:** `go.sum` is missing.

**Fix:**
```bash
go mod tidy       # generates go.sum locally
make docker-up    # now the COPY succeeds
```

### MongoDB replica set not initialised

**Cause:** `mongo-init` container ran before all three mongod nodes were healthy.

**Fix:**
```bash
docker compose down -v          # wipe volumes
docker compose up -d --build    # fresh start
```

The `mongo-init` service now guards against double-initialisation with
`try { rs.status() } catch(e) { rs.initiate(...) }`.

### Port already in use

```bash
# Find and free the port
lsof -ti:3000 | xargs kill -9
lsof -ti:27017 | xargs kill -9
lsof -ti:6379 | xargs kill -9
```

### App crashes immediately — JWT keys missing

```bash
make generate-keys    # creates ./keys/*.pem
make docker-rebuild   # rebuild + restart app only
```

---

## Development (hot reload)

```bash
# Install air (hot reload)
go install github.com/air-verse/air@latest

# Start local Mongo + Redis only
docker compose up -d mongo1 mongo2 mongo3 mongo-init redis

# Hot-reload API (reads .env automatically)
make dev
```

---

## Background Worker

The Asynq worker processes queued jobs (email, push notifications, cleanup):

```bash
# Start full stack including worker
make docker-worker

# Or run locally
make run-worker
```

---

## Seeded Credentials

| Email | Password | Role |
|-------|----------|------|
| admin@example.com | Admin@1234 | admin |
| user@example.com | User@1234 | user |

---

## Environment Variables Checklist

### Minimum (everything else has defaults)

```env
MONGO_URI=mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0
REDIS_HOST=redis
JWT_ACCESS_PRIVATE_KEY_PATH=./keys/access_private.pem
JWT_ACCESS_PUBLIC_KEY_PATH=./keys/access_public.pem
JWT_REFRESH_PRIVATE_KEY_PATH=./keys/refresh_private.pem
JWT_REFRESH_PUBLIC_KEY_PATH=./keys/refresh_public.pem
```

### Optional integrations

| Feature | Required vars |
|---------|--------------|
| Email | `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SES_FROM_EMAIL` |
| SMS / WhatsApp | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_SMS` |
| Push (FCM) | `FIREBASE_CREDENTIALS_FILE` |
| S3 Storage | `AWS_S3_BUCKET`, `AWS_REGION` |
| Google OAuth | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` |
| GitHub SSO | `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` |
| Apple Sign In | `APPLE_CLIENT_ID`, `APPLE_TEAM_ID`, `APPLE_KEY_ID`, `APPLE_PRIVATE_KEY_PATH` |
| Passkey | `WEBAUTHN_RP_ID`, `WEBAUTHN_RP_ORIGIN` |
| Sentry | `SENTRY_DSN` |
| Multi-tenant | `MULTI_TENANT_ENABLED=true`, `BASE_DOMAIN` |

---

## Make Target Reference

```
make setup            ← run this first on every new clone
make docker-up        ← start the full stack
make docker-down      ← stop everything
make seed             ← populate database
make migrate-indexes  ← create MongoDB indexes
make dev              ← hot-reload development
make test             ← unit tests
make test-e2e         ← end-to-end tests (stack must be running)
make test-coverage    ← HTML coverage report
make swagger          ← regenerate API docs
make lint             ← code quality check
make help             ← full target list
```

---

## Seeding Explained

`make seed` and `make migrate-indexes` work like this:

```
Host ──► docker run --network <bread-net> golang:1.23-alpine go run ./scripts/seed/main.go
                         │
                    bread-net network
                         │
              mongo1:27017  mongo2:27017  mongo3:27017
```

The `golang:1.23-alpine` container mounts your project directory, joins the
`bread-net` Docker network, and resolves MongoDB hostnames exactly as the app
container does.

### Local seed (no Docker, e.g. CI with a local MongoDB)

```bash
# Seed against localhost:27017 directly
make seed-local

# Or with a custom URI
MONGO_URI="mongodb://user:pass@myhost:27017/?authSource=admin" make seed-local
```

### Completely custom URI

```bash
MONGO_URI="mongodb://localhost:27017/?directConnection=true" \
MONGO_DB_NAME="mydb" \
go run ./scripts/seed/main.go
```
