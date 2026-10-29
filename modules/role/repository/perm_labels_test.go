package repository_test

// Guards the `permissions` block of locales/*.json — every grantable permission
// slug must have a human label, so a UI listing permissions (the per-user
// grant/revoke matrix) never shows a raw slug, and a slug added to
// AllAssignablePermissions without a label fails the build rather than leaking.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
)

// labelledLocales must define a label for every grantable permission. `id` is a
// partial locale that falls back to en per key, so only en is required here.
var labelledLocales = []string{"en"}

func permissionLabels(t *testing.T, lang string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "locales", lang+".json"))
	if err != nil {
		t.Fatalf("read %s locale: %v", lang, err)
	}
	var doc struct {
		Permissions map[string]string `json:"permissions"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse %s locale: %v", lang, err)
	}
	return doc.Permissions
}

func TestPermissionLabels_CoverEveryGrantablePermission(t *testing.T) {
	for _, lang := range labelledLocales {
		labels := permissionLabels(t, lang)
		var missing []string
		for _, p := range entity.AllAssignablePermissions() {
			if strings.TrimSpace(labels[string(p)]) == "" {
				missing = append(missing, string(p))
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("locales/%s.json is missing %d permission label(s):\n  %s",
				lang, len(missing), strings.Join(missing, "\n  "))
		}
	}
}

// A label must not be the slug itself — that defeats the dictionary.
func TestPermissionLabels_AreNotTheSlug(t *testing.T) {
	for _, lang := range labelledLocales {
		for slug, label := range permissionLabels(t, lang) {
			if label == slug {
				t.Errorf("locales/%s.json: %s labelled with its own slug", lang, slug)
			}
		}
	}
}
