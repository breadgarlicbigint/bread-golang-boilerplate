// Package apikey provides API key lifecycle — creation, validation, revocation.
//
// # Monolith usage
//
// Import the concrete service directly from modules/apikey/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package apikey
