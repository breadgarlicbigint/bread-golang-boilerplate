// Package storage abstracts file persistence behind a driver interface so the
// same callers work against S3 or the local filesystem (STORAGE_DRIVER=s3|local).
// Documents store the returned storage KEY, never a URL — resolve a key to a
// fetchable URL with PresignDownload at read time.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// Storage is the driver contract every backend satisfies.
type Storage interface {
	// Upload streams body under folder/filename and returns the storage key.
	Upload(ctx context.Context, folder, filename string, body io.Reader, contentType string) (key string, err error)
	// PresignUpload returns a URL the client can upload to directly plus the key
	// that upload will land at (S3: a signed PUT; local: the API's upload
	// endpoint — clients branch on the two-step vs direct flow).
	PresignUpload(ctx context.Context, folder, extension string) (uploadURL, key string, err error)
	// PresignDownload resolves a stored key to a fetchable URL. requestBase, when
	// non-empty, is "scheme://host" taken from the caller's own HTTP request and
	// takes precedence over any static config default — a dev port forward or
	// tunnel can remap the host/port the browser actually reaches, which no fixed
	// config value can predict. S3 ignores it: a presigned URL is already
	// absolute and independent of how the browser reached the API.
	PresignDownload(ctx context.Context, key string, requestBase string) (url string, err error)
	// Download opens the object at key for reading. Unlike PresignDownload it
	// hands back the BYTES rather than a URL, for server-side consumers that must
	// read an object the caller never sees. The caller closes the reader.
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object at key.
	Delete(ctx context.Context, key string) error
}

// New constructs the Storage driver selected by cfg.Storage.Driver. Defaults to
// the local filesystem driver (offline-friendly) when unset.
func New(cfg *config.Config) (Storage, error) {
	switch cfg.Storage.Driver {
	case "s3":
		return NewS3(cfg.AWS)
	case "local", "":
		return NewLocal(cfg.Storage)
	default:
		return nil, fmt.Errorf("storage: unknown driver %q (want s3|local)", cfg.Storage.Driver)
	}
}

// Ensure the concrete drivers satisfy the interface at compile time.
var (
	_ Storage = (*S3Storage)(nil)
	_ Storage = (*LocalStorage)(nil)
)
