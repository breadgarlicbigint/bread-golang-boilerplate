# ── Build stage ────────────────────────────────────────────────────────────────
FROM node:20-alpine AS email-builder

WORKDIR /email
COPY email-templates/package*.json ./
RUN npm ci --silent
COPY email-templates/ ./
# Stubs are the fallback if tsx fails
COPY pkg/email/dist/ /dist/
# Override output path so render.ts writes to /dist/
RUN EMAIL_DIST_DIR=/dist npx tsx src/render.ts 2>/dev/null || echo "  ⚠  Email build failed — using stubs"

# ── Go build stage ─────────────────────────────────────────────────────────────
FROM golang:1.23-alpine AS builder

ARG VERSION=dev

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy dependency manifests first for better layer caching
COPY go.mod go.sum ./
RUN go mod download || (go mod tidy && go mod download)

# Copy source
COPY . .

# Overwrite stub dist with React Email compiled output from the email-builder stage
COPY --from=email-builder /pkg-email-dist/ ./pkg/email/dist/

# Build — strip debug info
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
      -ldflags="-w -s -X main.Version=${VERSION}" \
      -o /app/server \
      ./apps/api

# ── Production image ───────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S appgroup && \
    adduser  -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/server        ./server
COPY --from=builder /app/locales       ./locales
COPY --from=builder /app/.env.example  ./.env.example

RUN mkdir -p ./keys

USER appuser

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- http://localhost:3000/health/live || exit 1

ENTRYPOINT ["/app/server"]
