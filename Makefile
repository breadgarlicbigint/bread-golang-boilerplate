.PHONY: all build run dev test test-unit test-integration test-e2e test-coverage \
        lint fmt vet tidy mod-sync ensure-modules deps \
        build-worker build-worker-rabbitmq build-worker-kafka \
        run-worker run-worker-rabbitmq run-worker-kafka \
        docker-up docker-down docker-logs docker-rebuild \
        docker-worker docker-rabbitmq docker-kafka \
        migrate-indexes migrate-indexes-local seed seed-local generate-keys swagger generate \
        build-emails web-install web-dev setup clean help

BINARY      := server
WORKER      := worker
BUILD_DIR   := ./build
CMD_API     := ./apps/api
CMD_WORKER  := ./apps/worker
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -ldflags "-w -s -X main.Version=$(VERSION)"

# ── Build ──────────────────────────────────────────────────────────────────────
all: lint test build

build: ensure-modules
	@echo "▶ Building $(BINARY) $(VERSION)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_API)

build-worker:
	@echo "▶ Building $(WORKER) (Redis/Asynq) $(VERSION)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(WORKER) $(CMD_WORKER)

build-worker-rabbitmq:
	@echo "▶ Building worker-rabbitmq $(VERSION)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/worker-rabbitmq ./apps/worker-rabbitmq

build-worker-kafka:
	@echo "▶ Building worker-kafka $(VERSION)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/worker-kafka ./apps/worker-kafka

run: ensure-modules build
	@$(BUILD_DIR)/$(BINARY)

run-worker: build-worker
	@$(BUILD_DIR)/$(WORKER)

run-worker-rabbitmq: build-worker-rabbitmq
	@$(BUILD_DIR)/worker-rabbitmq

run-worker-kafka: build-worker-kafka
	@$(BUILD_DIR)/worker-kafka

dev: ensure-modules
	@echo "▶ Starting hot-reload server (air)"
	@GOPATH=$$(go env GOPATH) ; \
	 if ! command -v air > /dev/null 2>&1 && [ ! -f "$$GOPATH/bin/air" ]; then \
	   echo "  Installing air..."; \
	   go install github.com/air-verse/air@latest; \
	 fi
	@PATH="$$(go env GOPATH)/bin:$$PATH" air -c .air.toml

# ensure-modules: self-heals go.sum if it's missing or stale.
# Runs automatically before 'dev', 'build', and 'run' so a fresh clone
# or an out-of-sync go.sum never blocks local development.
ensure-modules:
	@if [ ! -s go.sum ]; then \
	  echo "▶ go.sum missing or empty — running go mod tidy"; \
	  go mod tidy; \
	 fi

# ── Go modules ────────────────────────────────────────────────────────────────
tidy:
	@echo "▶ Running go mod tidy"
	go mod tidy

mod-sync: tidy
	@echo "▶ go.sum is up to date"

# deps: import/refresh the queue-driver client libraries (the RabbitMQ + Kafka
# "import command"). Pure-Go clients (no CGO) so builds stay CGO_ENABLED=0.
# Run after a fresh clone if go.sum is missing these, or to bump versions.
RABBITMQ_LIB := github.com/rabbitmq/amqp091-go@v1.10.0
KAFKA_LIB    := github.com/segmentio/kafka-go@v0.4.47

deps:
	@echo "▶ Importing queue driver dependencies (RabbitMQ + Kafka)"
	go get $(RABBITMQ_LIB)
	go get $(KAFKA_LIB)
	@$(MAKE) mod-sync
	@echo "✅  RabbitMQ + Kafka client libraries imported and go.sum synced"

# ── Test ──────────────────────────────────────────────────────────────────────
test:
	@echo "▶ Running all unit tests"
	go test ./... -v -race -count=1 -timeout 120s

test-unit:
	@echo "▶ Running unit tests (short)"
	go test ./... -v -race -short -count=1 -timeout 60s

test-integration:
	@echo "▶ Running integration tests (requires JWT keys)"
	go test ./tests/integration/... -v -race -count=1 -timeout 120s

test-e2e:
	@echo "▶ Running E2E tests (requires: make docker-up && make seed)"
	go test ./tests/e2e/... -v -count=1 -timeout 60s

test-coverage:
	@echo "▶ Coverage report"
	go test ./... -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report → coverage.html"

# ── Code quality ──────────────────────────────────────────────────────────────
lint:
	@echo "▶ Running golangci-lint"
	@GOPATH=$$(go env GOPATH) ; \
	 if ! command -v golangci-lint > /dev/null 2>&1 && [ ! -f "$$GOPATH/bin/golangci-lint" ]; then \
	   echo "  Installing golangci-lint..."; \
	   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$$GOPATH/bin"; \
	 fi
	@PATH="$$(go env GOPATH)/bin:$$PATH" golangci-lint run ./...

fmt:
	gofmt -w -s .

vet:
	go vet ./...

# ── Docker ────────────────────────────────────────────────────────────────────
docker-up: mod-sync
	@echo "▶ Starting full stack (API + MongoDB rs + Redis)"
	docker compose up -d --build
	@echo ""
	@echo "  ✅  API     → http://localhost:3000"
	@echo "  ✅  Health  → http://localhost:3000/health"
	@echo "  ✅  Docs    → http://localhost:3000/docs/index.html"
	@echo ""
	@echo "  Run 'make seed' to populate initial data (requires stack to be running)."

docker-worker: mod-sync
	@echo "▶ Starting full stack + Redis/Asynq worker"
	docker compose --profile worker up -d --build

docker-rabbitmq: mod-sync
	@echo "▶ Starting full stack + RabbitMQ broker + RabbitMQ worker"
	docker compose --profile rabbitmq up -d --build
	@echo "  ✅  RabbitMQ UI → http://localhost:15672 (guest/guest)"

docker-kafka: mod-sync
	@echo "▶ Starting full stack + Kafka broker + Kafka worker"
	docker compose --profile kafka up -d --build
	@echo "  ✅  Kafka broker → localhost:9094 (host) / kafka:9092 (in-network)"

docker-down:
	docker compose --profile worker --profile rabbitmq --profile kafka down

docker-stop:
	docker compose stop

docker-logs:
	docker compose logs -f app

docker-logs-worker:
	docker compose logs -f worker

docker-rebuild:
	docker compose up -d --build --force-recreate app

docker-clean:
	docker compose --profile worker --profile rabbitmq --profile kafka down -v --remove-orphans

# ── Database ──────────────────────────────────────────────────────────────────
# ── Database (Docker-aware) ───────────────────────────────────────────────────
# These targets run scripts INSIDE the Docker network so MongoDB service names
# (mongo1, mongo2, mongo3) resolve correctly.
# Prerequisites: make docker-up must be running first.

DOCKER_NETWORK := $(shell docker network ls --filter name=bread-net --format '{{.Name}}' 2>/dev/null | head -1)

migrate-indexes:
	@echo "▶ Creating MongoDB indexes (inside Docker network)"
	@test -n "$(DOCKER_NETWORK)" || (echo "❌  Docker stack not running. Run: make docker-up first" && exit 1)
	docker run --rm --network $(DOCKER_NETWORK) \
	    -v "$(CURDIR):/src" -w /src \
	    -e MONGO_URI="mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0" \
	    -e MONGO_DB_NAME="bread_boilerplate" \
	    golang:1.23-alpine \
	    sh -c "apk add --no-cache git ca-certificates > /dev/null 2>&1 && go run ./scripts/migrate/main.go"

seed:
	@echo "▶ Seeding database (inside Docker network)"
	@test -n "$(DOCKER_NETWORK)" || (echo "❌  Docker stack not running. Run: make docker-up first" && exit 1)
	docker run --rm --network $(DOCKER_NETWORK) \
	    -v "$(CURDIR):/src" -w /src \
	    -e MONGO_URI="mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0" \
	    -e MONGO_DB_NAME="bread_boilerplate" \
	    golang:1.23-alpine \
	    sh -c "apk add --no-cache git ca-certificates > /dev/null 2>&1 && go run ./scripts/seed/main.go"

# Run seed/migrate directly on host — loads .env, then applies overrides.
# Precedence (highest → lowest):
#   1. ENV=... variable passed on the command line  e.g. make seed-local MONGO_URI=...
#   2. .env file (loaded by the script via Viper)
#   3. Viper defaults (localhost:27017)
#
# Examples:
#   make seed-local                               # reads .env
#   make seed-local MONGO_URI=mongodb://prod:27017  # override URI only
ENV_FILE ?= .env

migrate-indexes-local:
	@echo "▶ Creating MongoDB indexes (reads $(ENV_FILE))"
	@[ -f "$(ENV_FILE)" ] && echo "  env file: $(ENV_FILE)" || echo "  ⚠  $(ENV_FILE) not found — using Viper defaults"
	BREAD_CONFIG_FILE=$(ENV_FILE) go run ./scripts/migrate/main.go

seed-local:
	@echo "▶ Seeding database (reads $(ENV_FILE))"
	@[ -f "$(ENV_FILE)" ] && echo "  env file: $(ENV_FILE)" || echo "  ⚠  $(ENV_FILE) not found — using Viper defaults"
	BREAD_CONFIG_FILE=$(ENV_FILE) go run ./scripts/seed/main.go

# ── Keys ──────────────────────────────────────────────────────────────────────
generate-keys:
	@echo "▶ Generating EC key pairs"
	@mkdir -p keys
	@echo "  Access token (ES256 — P-256)"
	openssl ecparam -genkey -name prime256v1 -noout | \
	  openssl pkcs8 -topk8 -nocrypt -out keys/access_private.pem
	openssl ec -in keys/access_private.pem -pubout -out keys/access_public.pem
	@echo "  Refresh token (ES512 — P-521)"
	openssl ecparam -genkey -name secp521r1 -noout | \
	  openssl pkcs8 -topk8 -nocrypt -out keys/refresh_private.pem
	openssl ec -in keys/refresh_private.pem -pubout -out keys/refresh_public.pem
	@echo "✅  Keys written to ./keys/"

# ── Swagger ───────────────────────────────────────────────────────────────────
# ── Email templates ───────────────────────────────────────────────────────────
build-emails:
	@echo "▶ Building React Email templates → pkg/email/dist/"
	@command -v node > /dev/null 2>&1 || (echo "❌  Node.js is required. Install from https://nodejs.org" && exit 1)
	@cd email-templates && \
	  ([ -d node_modules ] || npm install) && \
	  npm run build
	@echo "✅  Email templates compiled to pkg/email/dist/"

# ── Web test client ───────────────────────────────────────────────────────────
web-install:
	@echo "▶ Installing web test client dependencies → web/"
	@command -v node > /dev/null 2>&1 || (echo "❌  Node.js is required. Install from https://nodejs.org" && exit 1)
	@cd web && ([ -d node_modules ] || npm install)
	@echo "✅  Web test client dependencies installed"

web-dev: web-install
	@echo "▶ Starting web test client (Vite) on http://localhost:5173"
	@cd web && npm run dev -- --host 0.0.0.0

swagger:
	@echo "▶ Generating Swagger docs"
	@mkdir -p ./docs/swagger
	go run github.com/swaggo/swag/cmd/swag@v1.16.4 init \
	    -g $(CMD_API)/main.go \
	    -o ./docs/swagger \
	    --parseDependency \
	    --parseInternal
	@echo "✅  Swagger docs → ./docs/swagger/index.html"

# ── Mocks ─────────────────────────────────────────────────────────────────────
generate:
	@echo "▶ Generating mocks"
	go run go.uber.org/mock/mockgen@latest ./...
	go generate ./...

# ── First-time setup ──────────────────────────────────────────────────────────
setup:
	@echo "▶ Setting up project..."
	@echo ""
	@echo "Step 1/6 — Copying .env.example → .env"
	@cp -n .env.example .env 2>/dev/null && echo "  ✅  .env created" || echo "  ℹ️   .env already exists"
	@echo ""
	@echo "Step 2/6 — Generating EC key pairs"
	@$(MAKE) generate-keys
	@echo ""
	@echo "Step 3/6 — Building React Email templates"
	@$(MAKE) build-emails 2>/dev/null || echo "  ⚠️  Email build skipped — install Node.js and run: make build-emails"
	@echo ""
	@echo "Step 4/6 — Generating Swagger docs (must run BEFORE go mod tidy)"
	@$(MAKE) swagger || echo "  ⚠️  Swagger skipped — run manually: make swagger"
	@echo ""
	@echo "Step 5/6 — Syncing Go modules"
	@$(MAKE) mod-sync
	@echo ""
	@echo "Step 6/6 — Installing web test client dependencies"
	@$(MAKE) web-install 2>/dev/null || echo "  ⚠️  Web client install skipped — install Node.js and run: make web-install"
	@echo ""
	@echo "✅  Setup complete!"
	@echo ""
	@echo "  Next steps:"
	@echo "    1. Edit .env (add AWS keys, Twilio, Firebase, etc.)"
	@echo "    2. make docker-up"
	@echo "    3. make seed"
	@echo "    4. curl http://localhost:3000/health"
	@echo "    5. make web-dev   (test client → http://localhost:5173)"

# ── Helpers ───────────────────────────────────────────────────────────────────
clean:
	@rm -rf $(BUILD_DIR) coverage.out coverage.html docs/swagger/
	@echo "Cleaned."

help:
	@echo ""
	@echo "  Bread Golang Boilerplate — available targets"
	@echo ""
	@echo "  Setup"
	@echo "    make setup           First-time project setup (keys, .env, go.sum, swagger)"
	@echo "    make generate-keys   Generate EC key pairs for JWT"
	@echo "    make mod-sync        Run go mod tidy (generates/updates go.sum)"
	@echo ""
	@echo "  Development"
	@echo "    make dev             Hot reload with air"
	@echo "    make run             Build and run API server"
	@echo "    make run-worker          Build and run Redis/Asynq worker"
	@echo "    make run-worker-rabbitmq Build and run RabbitMQ worker"
	@echo "    make run-worker-kafka    Build and run Kafka worker"
	@echo "    make deps            Import RabbitMQ + Kafka client libs (go get + tidy)"
	@echo "    make swagger         Regenerate Swagger docs"
	@echo ""
	@echo "  Web test client"
	@echo "    make web-install     Install web/ (React) dependencies"
	@echo "    make web-dev         Run the web test client → http://localhost:5173"
	@echo ""
	@echo "  Docker"
	@echo "    make docker-up       Start API + MongoDB rs + Redis"
	@echo "    make docker-worker   Start full stack + Redis/Asynq worker"
	@echo "    make docker-rabbitmq Start full stack + RabbitMQ broker + worker"
	@echo "    make docker-kafka    Start full stack + Kafka broker + worker"
	@echo "    make docker-down     Stop and remove all containers"
	@echo "    make docker-logs     Tail app logs"
	@echo "    make docker-rebuild  Rebuild and restart only the app container"
	@echo "    make docker-clean    Remove containers AND volumes (destructive)"
	@echo ""
	@echo "  Database"
	@echo "    make migrate-indexes Create all MongoDB indexes"
	@echo "    make seed            Seed roles, users, feature flags, app versions"
	@echo ""
	@echo "  Testing"
	@echo "    make test            All unit tests"
	@echo "    make test-unit       Unit tests only (no DB/JWT required)"
	@echo "    make test-integration Integration tests (requires JWT keys)"
	@echo "    make test-e2e        E2E tests (requires running stack)"
	@echo "    make test-coverage   Coverage report → coverage.html"
	@echo ""
	@echo "  Quality"
	@echo "    make lint            Run golangci-lint"
	@echo "    make fmt             Format all Go files"
	@echo "    make vet             Run go vet"
	@echo ""
