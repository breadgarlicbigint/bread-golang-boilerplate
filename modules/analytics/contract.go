// Package analytics provides Analytics aggregations — user metrics, fraud signals, audit summaries.
//
// # Monolith usage
//
// Import the concrete service directly from modules/analytics/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers stay the same.
package analytics
