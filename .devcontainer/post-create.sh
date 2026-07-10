#!/usr/bin/env bash
# post-create.sh — runs ONCE after the dev container is created.
# Sets up keys, .env, email templates, swagger docs, and Go modules.

set -euo pipefail

WORKSPACE="/workspace"
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log()  { echo -e "${GREEN}▶ $*${NC}"; }
warn() { echo -e "${YELLOW}⚠  $*${NC}"; }
err()  { echo -e "${RED}✗  $*${NC}"; }

cd "$WORKSPACE"

echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║      Bread Golang Boilerplate — Dev Container Setup    ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""

# ── Step 1: .env ──────────────────────────────────────────────────────────────
log "Step 1/6 — Creating .env from .env.example"
if [ ! -f "$WORKSPACE/.env" ]; then
    cp "$WORKSPACE/.env.example" "$WORKSPACE/.env"
    echo "  .env created"
else
    echo "  .env already exists — skipping"
fi

# Override MongoDB + Redis to use devcontainer service names
# (these are always correct inside the devcontainer network)
sed -i 's|^MONGO_URI=.*|MONGO_URI=mongodb://mongo1:27017,mongo2:27017,mongo3:27017/?replicaSet=rs0|' "$WORKSPACE/.env"
sed -i 's|^REDIS_HOST=.*|REDIS_HOST=redis|' "$WORKSPACE/.env"
sed -i 's|^REDIS_PORT=.*|REDIS_PORT=6379|' "$WORKSPACE/.env"
echo "  Service URIs patched for devcontainer network"

# ── Step 2: EC key pairs ──────────────────────────────────────────────────────
log "Step 2/6 — Generating JWT EC key pairs"
mkdir -p "$WORKSPACE/keys"

if [ ! -f "$WORKSPACE/keys/access_private.pem" ]; then
    openssl ecparam -genkey -name prime256v1 -noout \
        | openssl pkcs8 -topk8 -nocrypt \
        -out "$WORKSPACE/keys/access_private.pem"
    openssl ec -in "$WORKSPACE/keys/access_private.pem" \
        -pubout -out "$WORKSPACE/keys/access_public.pem" 2>/dev/null
    echo "  Access token keys (ES256) generated"
else
    echo "  Access keys already exist — skipping"
fi

if [ ! -f "$WORKSPACE/keys/refresh_private.pem" ]; then
    openssl ecparam -genkey -name secp521r1 -noout \
        | openssl pkcs8 -topk8 -nocrypt \
        -out "$WORKSPACE/keys/refresh_private.pem"
    openssl ec -in "$WORKSPACE/keys/refresh_private.pem" \
        -pubout -out "$WORKSPACE/keys/refresh_public.pem" 2>/dev/null
    echo "  Refresh token keys (ES512) generated"
else
    echo "  Refresh keys already exist — skipping"
fi

# ── Step 3: React Email templates ─────────────────────────────────────────────
log "Step 3/6 — Building React Email templates"
if command -v node &> /dev/null; then
    cd "$WORKSPACE/email-templates"
    npm install --silent
    npm run build && echo "  Email templates compiled → pkg/email/dist/" \
        || warn "Email build failed — using stubs (run: make build-emails later)"
    cd "$WORKSPACE"
else
    warn "Node.js not found — using email stubs (run: make build-emails when available)"
fi

# ── Step 4: Swagger docs ──────────────────────────────────────────────────────
log "Step 4/6 — Generating Swagger documentation"
cd "$WORKSPACE"
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init \
    -g ./apps/api/main.go \
    -o ./docs/swagger \
    --parseDependency \
    --parseInternal \
    && echo "  Swagger docs generated → docs/swagger/" \
    || warn "Swagger generation failed — run: make swagger"

# ── Step 5: Go modules ────────────────────────────────────────────────────────
log "Step 5/6 — Syncing Go modules"
go mod tidy && echo "  go.sum updated" || err "go mod tidy failed"

# ── Step 6: Web test client ───────────────────────────────────────────────────
log "Step 6/6 — Installing web test client dependencies"
if command -v node &> /dev/null; then
    cd "$WORKSPACE/web"
    npm install --silent && echo "  Web test client dependencies installed" \
        || warn "npm install failed — run: make web-install later"
    cd "$WORKSPACE"
else
    warn "Node.js not found — run: make web-install when available"
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║  ✅  Dev container ready!                            ║"
echo "╠══════════════════════════════════════════════════════╣"
echo "║                                                      ║"
echo "║  Start dev server:   make dev                       ║"
echo "║  Start web client:   make web-dev                   ║"
echo "║  Seed database:      make seed                      ║"
echo "║  Run tests:          make test                      ║"
echo "║  View API docs:      http://localhost:3000/docs      ║"
echo "║  Web test client:    http://localhost:5173           ║"
echo "║                                                      ║"
echo "║  MongoDB:  mongo1:27017 (rs0)                       ║"
echo "║  Redis:    redis:6379                               ║"
echo "╚══════════════════════════════════════════════════════╝"
echo ""
