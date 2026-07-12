#!/usr/bin/env bash
# air-build-api.sh — build step run by `make dev` (.air.toml) on every hot
# reload: keeps Swagger docs and compiled email templates in sync with the
# source before rebuilding the API binary.
set -euo pipefail

STAMP="email-templates/.build-stamp"

# ── Email templates ──────────────────────────────────────────────────────────
# Only re-run the Node/npm build when a .tsx/.ts source file actually changed
# since the last build — most saves are Go-only and shouldn't pay this cost.
if command -v node > /dev/null 2>&1; then
  if [ ! -f "$STAMP" ] || [ -n "$(find email-templates/src -type f -newer "$STAMP" 2>/dev/null)" ]; then
    echo "▶ Email templates changed — rebuilding (make build-emails)"
    (cd email-templates && [ -d node_modules ] || npm install)
    (cd email-templates && npm run build)
    touch "$STAMP"
  fi
else
  echo "⚠  Node.js not found — skipping email template rebuild (pkg/email/dist/ stubs used as-is)"
fi

# ── Swagger docs ──────────────────────────────────────────────────────────────
echo "▶ Regenerating Swagger docs"
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init \
    -g ./apps/api/main.go \
    -o ./docs/swagger \
    --parseDependency \
    --parseInternal

# ── API binary ────────────────────────────────────────────────────────────────
go build -o .air/server ./apps/api
