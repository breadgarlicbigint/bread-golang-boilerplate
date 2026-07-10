// Package activity provides Bidirectional audit log — inbound requests + outbound side-effects.
//
// # Monolith usage
//
// Import the concrete service directly from modules/activity/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package activity
