# Changelog

All notable changes to this boilerplate are recorded here. This log tracks
changes ported back from downstream projects and other upstream improvements.

## Unreleased

### Added
- `MONGO_DEPLOYMENT=auto|standalone|replicaset|sharded` (default `auto`):
  detects whether multi-document transactions are available and runs
  `Transactor.WithTransaction` inside a non-transactional session on a
  standalone instead of failing. `Transactor.TransactionsEnabled()` /
  `database.NewTransactorWithMode` added.
- `pkg/storage` driver interface with a local-filesystem driver
  (`STORAGE_DRIVER=s3|local`, default `local`; `STORAGE_LOCAL_DIR`,
  `STORAGE_LOCAL_BASE_URL`, `UPLOAD_MAX_BYTES`). Runs offline with no cloud
  credentials.
- Error envelope now carries a stable machine `code` and an optional
  structured `details` payload; `errors.AppError` gains `Field`/`WithField`
  and `Details`/`WithDetails`.
- Validation messages are kind-aware and never leak identifiers; new
  `validation.*` keys and a `fields.<jsonTag>` label dictionary in
  `locales/en.json`, plus the `notblank` rule.

### Fixed
- Rate limiter no longer renews the window TTL on every request, so a client
  at the limit is admitted again after the window elapses (was blocked
  indefinitely).

### Changed
- Outbound channels (SMS, WhatsApp, FCM) now send only when their explicit
  `*_ENABLED` flag is set **and** credentials are present; otherwise they are
  simulated (logged, nothing sent). Default dev/tests run fully offline. Set
  `SMS_ENABLED`/`WHATSAPP_ENABLED`/`FCM_ENABLED` in an environment that should
  actually deliver.

### Changed (breaking)
- **`pkg/storage`**: `Storage.Upload(...)` now returns the **storage key**,
  not a public URL. Persist the key and resolve it to a URL at read time with
  `PresignDownload(ctx, key, requestBase)` — which gained the `requestBase`
  parameter (pass the caller's `scheme://host`; S3 ignores it). New
  `Download(ctx, key) (io.ReadCloser, error)` on both drivers.
  *Migration:* stop storing the `Upload` return value as a URL; store it as a
  key and call `PresignDownload` when you need a fetchable URL.
