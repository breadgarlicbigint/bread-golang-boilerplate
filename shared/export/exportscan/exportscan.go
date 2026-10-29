// Package exportscan finds every spreadsheet column and sheet declared in the
// tree, by parsing the source rather than by importing it.
//
// # Why it exists as its own package
//
// TWO guards need the same answer and must not drift apart:
//
//   - shared/export/labels_test.go asserts every declared column HAS a label in
//     every locale — without it a spreadsheet ships with a raw identifier in
//     row 1 (rule 14);
//   - shared/validate/field_labels_test.go asserts the `fields` block has no
//     ORPHANS — labels nothing uses any more. Since v1.67 that block is shared
//     between validation prose and export headers, so a check that only knew
//     about DTOs would report every export label as dead and demand its
//     deletion, which would then redden the first guard. The two are the same
//     dictionary read from two directions.
//
// A SOURCE SCAN rather than a registry the modules opt into, following
// shared/validate's own approach: a registry covers only what somebody
// remembered to register, and the declaration nobody thought to register is
// precisely the one these guards exist to catch.
//
// It is imported only by tests. It carries no runtime behaviour and nothing in
// the API binary references it.
package exportscan

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// Declarations is everything one scan found.
type Declarations struct {
	// LabelKeys are the `fields.`-prefixed keys every column's header resolves
	// through, deduplicated.
	LabelKeys []string
	// FieldNames are the same keys with the `fields.` prefix removed — the shape
	// the orphan check compares against DTO JSON tags.
	FieldNames map[string]bool
	// Sheets are the names handed to export.Respond, each of which needs a title
	// under `export.sheets`.
	Sheets []string
}

// Scan walks the given trees and returns every column and sheet declared in
// them. Test files are skipped: they declare throwaway fixtures that do not ship.
func Scan(roots ...string) (Declarations, error) {
	out := Declarations{FieldNames: map[string]bool{}}
	seenLabel := map[string]bool{}
	seenSheet := map[string]bool{}
	fset := token.NewFileSet()

	for _, root := range roots {
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
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return perr
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.CompositeLit:
					if !isColumnSlice(node.Type) {
						return true
					}
					for _, el := range node.Elts {
						lit, ok := el.(*ast.CompositeLit)
						if !ok {
							continue
						}
						name := columnFieldName(lit)
						if name == "" {
							continue
						}
						out.FieldNames[name] = true
						if key := "fields." + name; !seenLabel[key] {
							seenLabel[key] = true
							out.LabelKeys = append(out.LabelKeys, key)
						}
					}
				case *ast.CallExpr:
					if name, ok := respondSheet(node); ok && !seenSheet[name] {
						seenSheet[name] = true
						out.Sheets = append(out.Sheets, name)
					}
				}
				return true
			})
			return nil
		})
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// isColumnSlice matches `[]export.Column`. The elements inside a slice literal
// carry no type of their own, which is why the SLICE is what gets matched.
func isColumnSlice(expr ast.Expr) bool {
	arr, ok := expr.(*ast.ArrayType)
	if !ok {
		return false
	}
	sel, ok := arr.Elt.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Column" {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "export"
}

// columnFieldName returns the label NAME a column resolves through: its explicit
// Label, else the last segment of its dotted Field path. Mirrors
// export.Column.labelKey, which is unexported precisely because a column's
// header resolution is not a public contract.
func columnFieldName(lit *ast.CompositeLit) string {
	var field, label string
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		name, _ := kv.Key.(*ast.Ident)
		val, _ := kv.Value.(*ast.BasicLit)
		if name == nil || val == nil || val.Kind != token.STRING {
			continue
		}
		s, err := strconv.Unquote(val.Value)
		if err != nil {
			continue
		}
		switch name.Name {
		case "Field":
			field = s
		case "Label":
			label = s
		}
	}
	if label != "" {
		return label
	}
	if i := strings.LastIndex(field, "."); i >= 0 {
		return field[i+1:]
	}
	return field
}

// respondSheet matches `export.Respond(c, "customers", …)` and returns the sheet
// name, which needs a title in every locale.
func respondSheet(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Respond" {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "export" || len(call.Args) < 2 {
		return "", false
	}
	lit, ok := call.Args[1].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}
