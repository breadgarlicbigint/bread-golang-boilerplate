# Installation Guide

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.23+ | https://go.dev/dl |
| Docker | 28+ | https://docs.docker.com/get-docker |
| Docker Compose | v2 | bundled with Docker Desktop |
| OpenSSL | any | `brew install openssl` / `apt install openssl` |

---

## Option A — Docker (Recommended)

The fastest path to a running stack.

```bash
# 1. Clone the repo
git clone https://github.com/breadgarlicbigint/bread-golang-boilerplate
cd bread-golang-boilerplate

# 2. Bootstrap (generates keys, copies .env, tidies modules)
make setup

# 3. Review and fill in your .env values
#    At minimum: AWS keys if you want S3/SES; everything else has a default.
nano .env

# 4. Start MongoDB replica set (3 nodes) + Redis + App
make docker-up

# 5. Seed database (roles, users, feature flags)
make seed

# 6. Smoke test
curl http://localhost:3000/health
curl -X POST http://localhost:3000/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"Admin@1234"}'
```

The app auto-restarts on crash (`restart: unless-stopped`).

---

## Option B — Local (without Docker)

### 1. Start dependencies manually

**MongoDB replica set** (required for transactions):
```bash
# Start three mongod instances (or use Atlas free tier)
mongod --replSet rs0 --port 27017 --dbpath /tmp/mongo1 &
mongod --replSet rs0 --port 27018 --dbpath /tmp/mongo2 &
mongod --replSet rs0 --port 27019 --dbpath /tmp/mongo3 &

# Initiate the replica set
mongosh --eval 'rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "localhost:27017" },
    { _id: 1, host: "localhost:27018" },
    { _id: 2, host: "localhost:27019" }
  ]
})'
```

**Redis**:
```bash
redis-server
```

### 2. Install Go dependencies

```bash
go mod download
```

### 3. Generate EC key pairs

```bash
make generate-keys
# Writes keys/ directory (git-ignored)
```

### 4. Configure environment

```bash
cp .env.example .env
# Edit MONGO_URI, REDIS_HOST, JWT_*_KEY_PATH, etc.
```

### 5. Run database migrations (indexes) and seed

```bash
make migrate-indexes
make seed
```

### 6. Start the server

```bash
# Regular start
make run

# Hot reload (requires air)
make dev
```

---

## Environment Variables Reference

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `development` \| `staging` \| `production` |
| `APP_PORT` | `3000` | HTTP listen port |
| `MONGO_URI` | `mongodb://localhost:27017` | MongoDB connection string |
| `MONGO_DB_NAME` | `bread_boilerplate` | Database name |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `JWT_ACCESS_PRIVATE_KEY_PATH` | `./keys/access_private.pem` | ES256 private key |
| `JWT_ACCESS_EXPIRE` | `15m` | Access token TTL |
| `JWT_REFRESH_EXPIRE` | `168h` | Refresh token TTL (7 days) |
| `AUTH_MAX_PASSWORD_ATTEMPTS` | `5` | Failed logins before lockout |
| `AUTH_LOCKOUT_DURATION` | `15m` | Account lockout duration |
| `RATE_LIMIT_REQUESTS` | `100` | Requests per window per IP |
| `RATE_LIMIT_PERIOD` | `1m` | Rate limit window |
| `API_KEY_HEADER` | `X-Api-Key` | Header name for API keys |
| `AWS_REGION` | `us-east-1` | AWS region |
| `SENTRY_DSN` | _(empty)_ | Sentry DSN (disabled if empty) |
| `WORKER_CONCURRENCY` | `10` | Asynq worker goroutines |

---

## Seeded Credentials

After running `make seed`:

| Email | Password | Role |
|---|---|---|
| admin@example.com | Admin@1234 | admin |
| user@example.com | User@1234 | user |

---

## Production Checklist

- [ ] Set `APP_ENV=production` (disables Swagger)
- [ ] Use strong random `SESSION_SECRET`
- [ ] Mount EC keys as secrets, not baked into image
- [ ] Enable Redis TLS (`REDIS_TLS=true`)
- [ ] Use MongoDB Atlas or secured self-hosted replica set
- [ ] Set `SENTRY_DSN` for error tracking
- [ ] Configure AWS credentials via IAM role (not static keys)
- [ ] Review and tighten CORS `AllowOrigins`
- [ ] Set `RATE_LIMIT_REQUESTS` appropriate to your traffic
