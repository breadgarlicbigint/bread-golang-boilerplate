// Package role provides Role definitions, slugs, and permission constants.
//
// # Monolith usage
//
// Import the concrete service directly from modules/role/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package role

import (
	"context"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
)

// Service is the public interface for role lookups — the only thing other
// modules may import from this package.
type Service interface {
	List(ctx context.Context) ([]entity.Role, error)
}
