package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// LocalStorage writes objects to the local filesystem under Dir and serves them
// from BaseURL. It is the offline-friendly default driver — no cloud creds, so
// dev and the whole test suite run with zero external calls.
type LocalStorage struct {
	dir     string
	baseURL string
}

// NewLocal builds the filesystem driver, creating the root directory if needed.
func NewLocal(cfg config.StorageConfig) (*LocalStorage, error) {
	dir := cfg.LocalDir
	if dir == "" {
		dir = "./storage-data"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("storage/local: mkdir %q: %w", dir, err)
	}
	base := cfg.LocalBaseURL
	if base == "" {
		base = "http://localhost:3000/files"
	}
	return &LocalStorage{dir: dir, baseURL: strings.TrimRight(base, "/")}, nil
}

// Upload writes body to <dir>/<folder>/<filename> and returns the storage key
// (folder/filename), mirroring the S3 driver's key semantics.
func (l *LocalStorage) Upload(ctx context.Context, folder, filename string, body io.Reader, contentType string) (string, error) {
	key := buildKey(folder, filename)
	full := filepath.Join(l.dir, filepath.FromSlash(key))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", fmt.Errorf("storage/local: mkdir: %w", err)
	}
	f, err := os.Create(full)
	if err != nil {
		return "", fmt.Errorf("storage/local: create: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, body); err != nil {
		return "", fmt.Errorf("storage/local: write: %w", err)
	}
	return key, nil
}

// PresignUpload returns the API's direct-upload endpoint for the local driver
// (there is no signing) plus the key the upload will land at. Clients branch on
// driver: two-step signed PUT for S3, this direct POST for local.
func (l *LocalStorage) PresignUpload(ctx context.Context, folder, extension string) (string, string, error) {
	key := buildKey(folder, uuid.NewString()+extension)
	return l.baseURL + "/upload?key=" + key, key, nil
}

// PresignDownload resolves a key to its served URL. requestBase (e.g.
// "http://localhost:3001", no trailing slash), when given, overrides the
// configured baseURL — see the Storage interface doc for why a static value
// can't survive a forwarded/remapped port. Falls back to the configured baseURL
// for non-HTTP callers (e.g. a worker job with no request in scope).
func (l *LocalStorage) PresignDownload(ctx context.Context, key string, requestBase string) (string, error) {
	base := l.baseURL
	if requestBase != "" {
		base = strings.TrimRight(requestBase, "/") + "/files"
	}
	return base + "/" + strings.TrimLeft(key, "/"), nil
}

// Download opens the file at key for reading. The path is joined under the
// driver's root and cleaned first, so a key carrying "../" cannot escape it —
// keys are server-generated today, but this reads whatever it is handed.
func (l *LocalStorage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	full := filepath.Join(l.dir, filepath.FromSlash(path.Clean("/"+key)))
	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("storage/local: download %q: %w", key, err)
	}
	return f, nil
}

// Delete removes the file at key (a missing file is not an error).
func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	full := filepath.Join(l.dir, filepath.FromSlash(key))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("storage/local: delete: %w", err)
	}
	return nil
}
