// Package passkey provides WebAuthn passkey and biometric authentication ceremonies.
//
// # Monolith usage
//
// Import the concrete service directly from modules/passkey/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package passkey
