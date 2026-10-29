package handler_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The publish route must stay OUT of the published API spec (user requirement,
// 2026-09-11). `swag` documents only what is annotated, so the mechanism is an
// absence — and an absence is exactly the kind of thing someone restores while
// tidying up comments, with nothing failing.
//
// This is enforced two ways, deliberately. The AST scan is the real gate: it
// needs no generated artifact and so it means the same thing on a clean
// checkout and in CI. The swagger.json check is a second opinion that only runs
// where the file exists — `docs/swagger/*.json` is gitignored, so a test that
// relied on it alone would pass vacuously everywhere that matters.
//
// It is NOT a security boundary — the route is API-key gated . It is
// about not advertising a machine-only endpoint in the catalogue humans read.

// swaggerAnnotations are the comment directives `swag` acts on. `@Description`
// and friends are harmless alone, but `@Router` is what actually publishes a
// path, so all of them are refused rather than just the one.
var swaggerAnnotations = []string{"@Router", "@Summary", "@Tags", "@Accept", "@Produce", "@Success", "@Failure", "@Param", "@Security"}

func TestReleaseHandlerHasNoSwaggerAnnotations(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse handler package: %v", err)
	}

	found := false
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != "Release" || fn.Recv == nil {
					continue
				}
				found = true
				if fn.Doc == nil {
					continue
				}
				for _, c := range fn.Doc.List {
					for _, ann := range swaggerAnnotations {
						if strings.Contains(c.Text, ann) {
							t.Errorf("Release carries the swagger annotation %s — the publish route "+
								"must stay out of the generated spec \n  %s",
								ann, strings.TrimSpace(c.Text))
						}
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("no Release method found on the handler — did it move? This guard must move with it")
	}
}

func TestReleasePathAbsentFromGeneratedSpec(t *testing.T) {
	// `docs/swagger/docs.go` and NOT `swagger.json`: the .json/.yaml outputs are
	// gitignored, so a test reading those passes vacuously on a fresh checkout
	// and in CI — which is everywhere it would matter. docs.go is committed, so
	// this runs for real, every time.
	path := filepath.Join("..", "..", "..", "docs", "swagger", "docs.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated spec: %v", err)
	}
	spec := string(raw)
	// Sanity check first: an empty or truncated file would make the assertion
	// below pass by saying nothing at all.
	if !strings.Contains(spec, "/v1/app-version/check") {
		t.Fatal("generated spec does not contain the version-check route — regenerate with `make swagger`; " +
			"without it the assertion below proves nothing")
	}
	if strings.Contains(spec, "app-version/release") {
		t.Error("the publish route is present in docs/swagger/docs.go and must not be ")
	}
}
