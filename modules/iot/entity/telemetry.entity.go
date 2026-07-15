package entity

import (
	"time"

	"github.com/google/uuid"
)

// DeviceTelemetry is a single reading received on the "devices/+/telemetry"
// MQTT topic, persisted for GET /v1/admin/iot/telemetry. TTL-indexed (1
// day) — this is a diagnostic feed for the demo, not long-term storage; the
// live view is the realtime WS/SSE push, this is just "what did I miss".
type DeviceTelemetry struct {
	ID         uuid.UUID `bson:"_id,omitempty" json:"id"`
	DeviceID   string    `bson:"deviceId"      json:"deviceId"`
	Metric     string    `bson:"metric"        json:"metric"`
	Value      float64   `bson:"value"         json:"value"`
	Unit       string    `bson:"unit"          json:"unit,omitempty"`
	RecordedAt time.Time `bson:"recordedAt"    json:"recordedAt"`
}
