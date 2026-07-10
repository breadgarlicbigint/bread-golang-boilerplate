// swagger.go — blank import so swag's init() registers the spec with gin-swagger.
// This file is the ONLY place the generated docs package is imported.
// 'make swagger' regenerates docs/swagger/docs.go; this import wires it in.
//
// go mod tidy will NOT get stuck on this because cmd/api is compiled
// after docs/swagger/docs.go is known to exist locally (same module).
package main

import _ "github.com/breadgarlicbigint/bread-golang-boilerplate/docs/swagger"
