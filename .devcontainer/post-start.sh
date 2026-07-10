#!/usr/bin/env bash
# post-start.sh — runs EVERY TIME the dev container starts (after restarts too).
# Lighter than post-create: just ensures services are healthy.

set -uo pipefail

YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m'

log()  { echo -e "${GREEN}▶ $*${NC}"; }
warn() { echo -e "${YELLOW}⚠  $*${NC}"; }

cd /workspace

# ── Wait for MongoDB ──────────────────────────────────────────────────────────
log "Waiting for MongoDB replica set..."
RETRIES=20
until mongosh --host mongo1:27017 --eval "db.adminCommand('ping').ok" --quiet 2>/dev/null | grep -q "1"; do
    RETRIES=$((RETRIES - 1))
    if [ $RETRIES -le 0 ]; then
        warn "MongoDB not ready after 60s — services may not be running"
        break
    fi
    sleep 3
done

# Check replica set status
RS_STATUS=$(mongosh --host mongo1:27017 --eval "rs.status().ok" --quiet 2>/dev/null || echo "0")
if [ "$RS_STATUS" = "1" ]; then
    echo "  MongoDB rs0 ready ✅"
else
    warn "Replica set not initialised — run: make migrate-indexes"
fi

# ── Wait for Redis ────────────────────────────────────────────────────────────
log "Checking Redis..."
if redis-cli -h redis -p 6379 ping 2>/dev/null | grep -q "PONG"; then
    echo "  Redis ready ✅"
else
    warn "Redis not ready — check devcontainer services"
fi

echo ""
echo "Dev container running — type 'make dev' to start the API server"
