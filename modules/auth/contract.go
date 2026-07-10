// Package auth provides authentication, session management, and token issuance.
//
// # Monolith usage
//
// Import the concrete service directly:
//
//	import authsvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/service"
//
// # Microservice extraction
//
// Replace the concrete service with an HTTP/gRPC client that implements
// the Service interface below. All callers remain unchanged.
package auth

import "context"

// Service is the public contract other modules use to interact with auth.
// When extracting auth to a standalone microservice, provide an HTTP or
// gRPC client that satisfies this interface — all call-sites stay the same.
type Service interface {
	// ValidateAccessToken verifies an access token and returns its claims.
	ValidateAccessToken(ctx context.Context, token string) (*Claims, error)
}

// Claims is the minimal user data other modules need from a validated token.
type Claims struct {
	UserID    string // UUID or ObjectID hex string
	SessionID string
	Role      string // slug: "admin", "user", "member"
}
