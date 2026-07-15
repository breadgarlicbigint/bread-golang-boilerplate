package dto

// SimulateRequest triggers a simulated device reading — the API publishes
// to MQTT topic devices/:deviceId/telemetry exactly as a real device would,
// so the full MQTT round trip (publish → subscribed handler → Mongo →
// realtime push) is exercisable from the web test client without needing
// an actual device or MQTT client. Value is randomized within a plausible
// range for Metric when omitted.
type SimulateRequest struct {
	Metric string   `json:"metric" validate:"required"`
	Value  *float64 `json:"value"`
	Unit   string   `json:"unit"`
}

// CommandRequest publishes to devices/:deviceId/commands — fire-and-forget;
// nothing in this boilerplate subscribes to it (a real device would).
type CommandRequest struct {
	Command string         `json:"command" validate:"required"`
	Data    map[string]any `json:"data"`
}

type SimulateResponse struct {
	Published bool `json:"published"`
}

type CommandResponse struct {
	Published bool `json:"published"`
}

type TelemetryResponse struct {
	ID         string  `json:"id"`
	DeviceID   string  `json:"deviceId"`
	Metric     string  `json:"metric"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit,omitempty"`
	RecordedAt string  `json:"recordedAt"`
}
