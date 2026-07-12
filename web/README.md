# ACK API Test Console

A React + TypeScript SPA that exercises every HTTP-exposed feature of the Go
API (`apps/api`) — auth, 2FA, passkeys/WebAuthn, mobile OTP, notifications,
admin user management, app versioning, admin analytics, and admin
notifications (test email / send / broadcast — the page for exercising
transactional-vs-promotional queue routing) — plus a generic API console and
client-side activity log for anything not covered by a dedicated page.

## Running

### Inside the dev container (recommended)

Dependencies are installed automatically by `.devcontainer/post-create.sh`.
From a terminal in the container:

```bash
make dev       # starts the Go API on :3000 (separate terminal)
make web-dev   # starts this app on :5173
```

Both ports are forwarded automatically by the dev container. Open
`http://localhost:5173`.

### Standalone

```bash
cd web
npm install
cp .env.example .env   # VITE_API_BASE_URL, defaults to http://localhost:3000
npm run dev
```

## Session handling

Every authenticated request that gets a `401` first tries a silent token
refresh (`POST /v1/auth/refresh`). If that also fails — meaning the session
is genuinely invalid (expired, revoked via Logout-all, or missing) — the app
clears the stored session, shows a toast, and redirects to `/login` from
wherever you are, not just pages behind a protected route. Other error
statuses (400/403/404/409/422/500) are left alone and just surface in the
page's error panel — they don't log you out. Manually clicking **Logout**
also redirects to `/login`, without the "session expired" toast. See
`lib/apiClient.ts` (the 401/refresh branch) and `components/SessionWatcher.tsx`.

## Configuration

- `VITE_API_BASE_URL` (build-time, `.env`) — default API base URL, editable
  at runtime from the in-app **Settings** page (stored in `localStorage`).
- The **Settings** page also controls the `x-custom-lang`, `X-Tenant-ID`,
  `X-App-Version`, and `X-App-Platform` headers sent on every request.

## Passkeys / WebAuthn

Passkey registration uses the browser's real `navigator.credentials` API via
`@simplewebauthn/browser`. For ceremonies to succeed, the API's
`WEBAUTHN_RP_ORIGIN` must equal the exact origin this app is served from
(`http://localhost:5173` in the dev container / local setup — already the
default in `.env.example`). If you serve this app from a different host or
port, update `WEBAUTHN_RP_ORIGIN` accordingly and restart the API.

Two passkey login flows are wired to endpoints that are incomplete in the
current backend (usernameless/discoverable login always fails server-side;
identified login is an unwired stub) — see the in-app notes on the Passkeys
page. Passkey **registration** and **listing/deletion** are fully functional.

## GitHub / Apple OAuth

The GitHub callback (`GET /v1/auth/github/callback`) returns the login JSON
directly rather than redirecting back into this SPA, so "Login with GitHub"
opens it in a new tab — copy the JSON body and use the "Import session from
pasted JSON" box on the OAuth page. Apple Sign In requires a registered
Apple Service ID and can't be triggered from an arbitrary dev origin; the
OAuth page exposes a raw form to POST an existing `code`/`id_token` pair
directly to `/v1/auth/apple/callback`.
