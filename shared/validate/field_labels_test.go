package validate_test

// Guards the `fields` block of locales/*.json — the dictionary that lets a
// validation message name a field without exposing its DTO/storage identifier.
//
// This exists because the leak returns silently otherwise: add a DTO field with
// a `validate:` tag and no label, and shared/validate falls back to humanising
// the JSON tag ("Role id", "Refresh token"). That reads plausible enough to
// ship, which is exactly how raw names reach users. A missing label is a build
// failure, not a follow-up.

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/export/exportscan"
)

// Trees scanned for validated DTO fields. Every request struct the API binds
// lives under one of these.
var dtoRoots = []string{
	filepath.Join("..", "..", "modules"),
	filepath.Join("..", "..", "pkg"),
	filepath.Join("..", "..", "shared"),
}

// labelledLocales must define a label for every validated field. `id` is a
// partial locale (pkg/i18n/locales_repo_test.go) and falls back to en per key,
// so it is deliberately not required here.
var labelledLocales = []string{"en"}

// validatedFieldNames returns every JSON tag attached to a field that also
// carries a `validate:` tag, across the DTO trees. Nested input structs are
// included — go-playground reports the leaf field name, so they need labels too.
func validatedFieldNames(t *testing.T) map[string][]string {
	t.Helper()

	out := map[string][]string{}
	for _, root := range dtoRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" || d.Name() == "node_modules" {
					return fs.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			ast.Inspect(file, func(n ast.Node) bool {
				st, ok := n.(*ast.StructType)
				if !ok || st.Fields == nil {
					return true
				}
				for _, f := range st.Fields.List {
					if f.Tag == nil {
						continue
					}
					raw, err := strconv.Unquote(f.Tag.Value)
					if err != nil {
						continue
					}
					tag := reflect.StructTag(raw)
					if tag.Get("validate") == "" {
						continue
					}
					name := strings.Split(tag.Get("json"), ",")[0]
					if name == "" || name == "-" {
						continue
					}
					out[name] = append(out[name], path)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	if len(out) == 0 {
		t.Fatal("found no validated DTO fields — the scan is broken, not the locales")
	}
	return out
}

// fieldLabels reads the `fields` block of a locale file.
func fieldLabels(t *testing.T, lang string) map[string]string {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("..", "..", "locales", lang+".json"))
	if err != nil {
		t.Fatalf("read %s locale: %v", lang, err)
	}
	var doc struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse %s locale: %v", lang, err)
	}
	return doc.Fields
}

// The guard: every validated field must have a human label to be named by.
func TestFieldLabels_CoverEveryValidatedDTOField(t *testing.T) {
	fields := validatedFieldNames(t)

	for _, lang := range labelledLocales {
		labels := fieldLabels(t, lang)
		var missing []string
		for name, files := range fields {
			if strings.TrimSpace(labels[name]) == "" {
				sort.Strings(files)
				missing = append(missing, name+"  (e.g. "+files[0]+")")
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("locales/%s.json is missing %d field label(s) — a validation message would fall back to the raw identifier:\n  %s",
				lang, len(missing), strings.Join(missing, "\n  "))
		}
	}
}

// The dictionary must not rot either: a label for a field that no DTO validates
// AND no export column uses is dead weight that hides real drift. The `fields`
// block has two consumers — validation messages and spreadsheet export headers
// (shared/export) — so the orphan check must know about both or it would demand
// deleting every export-only label.
func TestFieldLabels_HaveNoOrphans(t *testing.T) {
	fields := validatedFieldNames(t)

	exported, err := exportscan.Scan(dtoRoots...)
	if err != nil {
		t.Fatalf("scan export columns: %v", err)
	}

	for _, lang := range labelledLocales {
		var orphans []string
		for name := range fieldLabels(t, lang) {
			if _, ok := fields[name]; ok {
				continue
			}
			if exported.FieldNames[name] {
				continue
			}
			orphans = append(orphans, name)
		}
		sort.Strings(orphans)
		if len(orphans) > 0 {
			t.Errorf("locales/%s.json labels %d field(s) that no DTO validates and no export column uses — drop them:\n  %s",
				lang, len(orphans), strings.Join(orphans, "\n  "))
		}
	}
}

// A label that is itself the identifier defeats the point of the dictionary.
func TestFieldLabels_AreNotTheIdentifier(t *testing.T) {
	for _, lang := range labelledLocales {
		for name, label := range fieldLabels(t, lang) {
			if label == name {
				t.Errorf("locales/%s.json: %s is labelled with its own JSON tag %q", lang, name, label)
			}
		}
	}
}
