package export_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/export/exportscan"
)

// Every export column's header and every sheet's title must resolve in the
// default locale, or a spreadsheet ships with a raw identifier in row 1.
//
// The declarations are found by SOURCE SCAN (shared/export/exportscan) rather
// than a registry modules opt into — a column added months from now beside an
// existing one is exactly the case a registry would forget. A column reuses the
// `fields` block validation prose already requires, so a new column whose field
// is already validated costs no new key. (`id` is a partial locale that falls
// back to en per key, so only en is required here.)

func localeDoc(t *testing.T, lang string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "locales", lang+".json"))
	if err != nil {
		t.Fatalf("read %s locale: %v", lang, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse %s locale: %v", lang, err)
	}
	return doc
}

func has(doc map[string]any, dotted string) bool {
	cur := any(doc)
	for _, seg := range strings.Split(dotted, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		cur, ok = m[seg]
		if !ok {
			return false
		}
	}
	s, ok := cur.(string)
	return ok && s != ""
}

func TestEveryExportLabelResolvesInEnglish(t *testing.T) {
	found, err := exportscan.Scan(
		filepath.Join("..", "..", "modules"),
		filepath.Join("..", "..", "shared"),
		filepath.Join("..", "..", "apps"),
	)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(found.LabelKeys) == 0 || len(found.Sheets) == 0 {
		t.Fatalf("scanned %d columns and %d sheets — the scan found nothing, which means it is no longer looking where the declarations live",
			len(found.LabelKeys), len(found.Sheets))
	}

	doc := localeDoc(t, "en")
	var missing []string
	for _, key := range found.LabelKeys {
		if !has(doc, key) {
			missing = append(missing, key)
		}
	}
	for _, name := range found.Sheets {
		if key := "export.sheets." + name; !has(doc, key) {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("en.json is missing %d export key(s) — a spreadsheet would ship with an identifier in row 1:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}
