// Package user provides User management — CRUD, profile, password, block/unblock.
//
// # Monolith usage
//
// Import the concrete service directly from modules/user/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package user

import "context"

// Service is the public contract other modules use.
type Service interface {
	GetByID(ctx context.Context, id string) (*UserInfo, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// UserInfo is the minimal user data other modules need.
type UserInfo struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	Status    string
	RoleID    string
}
