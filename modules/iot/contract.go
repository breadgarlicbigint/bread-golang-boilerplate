// Package iot is a device-telemetry demo built on MQTT — separate from and
// independent of the modules/notification / modules/realtime task queue and
// pub/sub systems. It exists to exercise a real MQTT publish/subscribe round
// trip end-to-end: POST /v1/admin/iot/devices/:deviceId/simulate publishes a
// reading to the broker as if a device sent it; a subscriber running inside
// this same process picks it up, persists it, and forwards it onto
// modules/realtime's generic pub/sub topic so a connected WebSocket/SSE
// client watching "iot:telemetry" sees it arrive live.
//
// # Monolith usage
//
// Import the concrete service directly from modules/iot/service.
//
// # Microservice extraction
//
// Implement the Service interface with an HTTP/gRPC client — all callers
// stay the same. Note the MQTT subscription itself (Start) is not part of
// this contract — it's an internal detail of whichever process owns the
// broker connection.
package iot

import "context"

// Service is the public contract other modules use.
type Service interface {
	// SimulateTelemetry publishes a reading to devices/:deviceId/telemetry.
	// value is nil to let the service pick a plausible random value for metric.
	SimulateTelemetry(ctx context.Context, deviceID, metric string, value *float64, unit string) error
	// SendCommand publishes to devices/:deviceId/commands.
	SendCommand(ctx context.Context, deviceID, command string, data map[string]any) error
}
