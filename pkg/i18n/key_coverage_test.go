package i18n_test

// Every i18n key the Go code names must resolve in `locales/en.json`.
//
// The lockstep test next door proves parity between locales, so a key missing
// from EVERY bundle is perfectly in lockstep and sails straight through —
// rendering as the raw key, which looks like a styling oddity rather than a
// missing translation (the translator returns the key when it cannot resolve it).
//
// Scope and limits: only STATIC string literals are seen. A key built at the
// call site is invisible here.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Every locale that must carry a translation for a key the code names. `id` is a
// partial bundle (see locales_repo_test.go) and falls back to en per key, so it
// is deliberately not required here.
var keyedLocales = []string{"en"}

var (
	// errors.NewI18n(status, "CODE", "i18n.key", "fallback")
	newI18nRe = regexp.MustCompile(`NewI18n\(\s*[^,]+,\s*"[^"]*"\s*,\s*"([^"]+)"`)
	// response.OKI18n(c, "i18n.key", …) and its siblings. ErrorI18n takes a
	// status first (ErrorI18n(c, http.StatusX, "key")), so an optional
	// non-quoted token is allowed between `c,` and the key.
	respI18nRe = regexp.MustCompile(`(?:OKI18n|CreatedI18n|ErrorI18n|OKWithMetaI18n|UnauthorizedI18n|ForbiddenI18n|NotFoundI18n)\(\s*c\s*,\s*(?:[^,"]+,\s*)?"([^"]+)"`)
)

func repoRoot() string { return filepath.Join("..", "..") }

// loadLocale reads one bundle into dot-separated key → value pairs.
func loadLocale(t *testing.T, lang string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(), "locales", lang+".json"))
	if err != nil {
		t.Fatalf("read %s locale: %v", lang, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse %s locale: %v", lang, err)
	}
	out := map[string]string{}
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			switch val := v.(type) {
			case string:
				out[key] = val
			case map[string]any:
				walk(key, val)
			}
		}
	}
	walk("", doc)
	return out
}

// referencedKeys walks the Go tree for statically-named i18n keys.
func referencedKeys(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{} // key → first file that names it
	roots := []string{"modules", "shared", "apps", "pkg"}
	for _, root := range roots {
		err := filepath.Walk(filepath.Join(repoRoot(), root), func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			b, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, re := range []*regexp.Regexp{newI18nRe, respI18nRe} {
				for _, m := range re.FindAllStringSubmatch(string(b), -1) {
					if _, seen := out[m[1]]; !seen {
						out[m[1]] = path
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	return out
}

func TestI18nKeys_EveryKeyTheCodeNamesResolves(t *testing.T) {
	keys := referencedKeys(t)
	if len(keys) < 40 {
		// The scan finding almost nothing would make this test pass vacuously
		// forever — the failure mode of every regex-based guard.
		t.Fatalf("only %d i18n keys found in the Go tree; the scanner is probably broken", len(keys))
	}

	for _, lang := range keyedLocales {
		bundle := loadLocale(t, lang)
		var missing []string
		for key, file := range keys {
			if _, ok := bundle[key]; !ok {
				rel, _ := filepath.Rel(repoRoot(), file)
				missing = append(missing, key+"  ("+rel+")")
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("locales/%s.json is missing %d key(s) the code names — the message would render as the key itself:\n  %s",
				lang, len(missing), strings.Join(missing, "\n  "))
		}
	}
}
