// Package mobile provides Mobile number verification via SMS and WhatsApp OTP.
//
// # Monolith usage
//
// Import the concrete service directly from modules/mobile/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package mobile
