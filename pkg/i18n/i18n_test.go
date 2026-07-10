package i18n_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
)

func setupLocales(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	en := `{
		"auth": { "loginSuccess": "Login successful", "invalidCredentials": "Invalid email or password" },
		"user": { "notFound": "User not found" },
		"validation": { "required": "{{.Field}} is required" }
	}`
	id := `{
		"auth": { "loginSuccess": "Login berhasil" },
		"user": {}
	}`

	if err := os.WriteFile(filepath.Join(dir, "en.json"), []byte(en), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "id.json"), []byte(id), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNew_LoadsLocales(t *testing.T) {
	dir := setupLocales(t)
	tr, err := i18n.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	langs := tr.SupportedLanguages()
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}

func TestNew_MissingDefaultLocale(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "fr.json"), []byte(`{"auth":{}}`), 0644)
	_, err := i18n.New(dir)
	if err == nil {
		t.Error("expected error when default locale (en) is missing")
	}
}

func TestT_ExactKey(t *testing.T) {
	dir := setupLocales(t)
	tr, _ := i18n.New(dir)

	got := tr.T("en", "auth.loginSuccess")
	if got != "Login successful" {
		t.Errorf("want 'Login successful', got %q", got)
	}
}

func TestT_FallbackToDefault(t *testing.T) {
	dir := setupLocales(t)
	tr, _ := i18n.New(dir)

	// "user.notFound" exists in en but not id
	got := tr.T("id", "user.notFound")
	if got != "User not found" {
		t.Errorf("expected fallback to en, got %q", got)
	}
}

func TestT_IndonesianOverride(t *testing.T) {
	dir := setupLocales(t)
	tr, _ := i18n.New(dir)

	got := tr.T("id", "auth.loginSuccess")
	if got != "Login berhasil" {
		t.Errorf("expected Indonesian translation, got %q", got)
	}
}

func TestT_MissingKeyReturnsKey(t *testing.T) {
	dir := setupLocales(t)
	tr, _ := i18n.New(dir)

	got := tr.T("en", "nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("expected key passthrough, got %q", got)
	}
}

func TestT_Interpolation(t *testing.T) {
	dir := setupLocales(t)
	tr, _ := i18n.New(dir)

	got := tr.T("en", "validation.required", map[string]string{"Field": "Email"})
	if got != "Email is required" {
		t.Errorf("expected interpolated string, got %q", got)
	}
}

func TestT_UnknownLanguageFallsBack(t *testing.T) {
	dir := setupLocales(t)
	tr, _ := i18n.New(dir)

	got := tr.T("xx", "auth.loginSuccess")
	if got != "Login successful" {
		t.Errorf("unknown lang should fall back to en, got %q", got)
	}
}
