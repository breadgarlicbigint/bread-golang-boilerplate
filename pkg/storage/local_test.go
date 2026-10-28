package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// The local driver uploads, resolves and downloads a file entirely offline —
// no cloud creds, no network.
func TestLocalStorageRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocal(config.StorageConfig{LocalDir: dir, LocalBaseURL: "http://localhost:3000/files/"})
	if err != nil {
		t.Fatal("new local:", err)
	}
	ctx := context.Background()

	key, err := s.Upload(ctx, "avatars", "photo.png", bytes.NewBufferString("hello"), "image/png")
	if err != nil {
		t.Fatal("upload:", err)
	}
	if key != "avatars/photo.png" {
		t.Errorf("unexpected key: %q", key)
	}
	// The bytes landed under the configured root.
	if b, err := os.ReadFile(filepath.Join(dir, "avatars", "photo.png")); err != nil || string(b) != "hello" {
		t.Errorf("file not written: %q err=%v", b, err)
	}

	rc, err := s.Download(ctx, key)
	if err != nil {
		t.Fatal("download:", err)
	}
	got, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(got) != "hello" {
		t.Errorf("download = %q, want hello", got)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatal("delete:", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "avatars", "photo.png")); !os.IsNotExist(err) {
		t.Error("file not deleted")
	}
	// Deleting a missing key is not an error.
	if err := s.Delete(ctx, key); err != nil {
		t.Errorf("second delete should be a no-op, got %v", err)
	}
}

// requestBase overrides the configured base so a forwarded/remapped port
// resolves to somewhere the caller can actually reach; a blank one falls back.
func TestLocalPresignDownload_RequestBaseOverridesConfig(t *testing.T) {
	s, err := NewLocal(config.StorageConfig{LocalDir: t.TempDir(), LocalBaseURL: "http://localhost:3000/files"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	url, _ := s.PresignDownload(ctx, "avatars/photo.png", "")
	if url != "http://localhost:3000/files/avatars/photo.png" {
		t.Errorf("default base: %q", url)
	}

	url, _ = s.PresignDownload(ctx, "avatars/photo.png", "http://localhost:3001/")
	if !strings.HasPrefix(url, "http://localhost:3001/files/") {
		t.Errorf("requestBase not honoured: %q", url)
	}
}
