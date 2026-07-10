// Package appversion provides App version policy — checks client version, returns update requirements.
//
// # Monolith usage
//
// Import the concrete service directly from modules/appversion/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package appversion
