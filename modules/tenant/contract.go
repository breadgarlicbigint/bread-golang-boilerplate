// Package tenant provides Multi-tenant resolution — maps request headers/subdomains to tenants.
//
// # Monolith usage
//
// Import the concrete service directly from modules/tenant/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package tenant

import "context"

// Service is the public contract other modules use.
type Service interface {
	FindBySlug(ctx context.Context, slug string) (*TenantInfo, error)
	FindByDomain(ctx context.Context, domain string) (*TenantInfo, error)
}

// TenantInfo is the minimal tenant data other modules need.
type TenantInfo struct {
	ID     string
	Slug   string
	Status string
}
