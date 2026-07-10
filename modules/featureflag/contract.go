// Package featureflag provides Feature flag evaluation — boolean toggles with rollout percentage.
//
// # Monolith usage
//
// Import the concrete service directly from modules/featureflag/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package featureflag

import "context"

// Service is the public contract other modules use.
type Service interface {
	IsEnabled(ctx context.Context, key string) (bool, error)
}
