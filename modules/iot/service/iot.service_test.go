package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.uber.org/zap"

	mqttpkg "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/mqtt"
)

// fakeMQTT records Publish calls and satisfies MQTTClient without a real
// broker. Subscribe is never exercised by these tests (Start/handleTelemetry
// need a real Mongo collection — see the package doc comment on why that's
// out of scope for unit tests here, same as modules/notification/service).
type fakeMQTT struct {
	published []publishedMsg
	pubErr    error
}

type publishedMsg struct {
	topic   string
	payload []byte
}

func (f *fakeMQTT) Publish(topic string, payload []byte) error {
	if f.pubErr != nil {
		return f.pubErr
	}
	f.published = append(f.published, publishedMsg{topic: topic, payload: payload})
	return nil
}

func (f *fakeMQTT) Subscribe(_ string, _ mqttpkg.MessageHandler) error { return nil }

var _ MQTTClient = (*fakeMQTT)(nil)

func TestSimulateTelemetryPublishesToDeviceTopic(t *testing.T) {
	mqtt := &fakeMQTT{}
	svc := &IoTService{mqtt: mqtt, log: zap.NewNop()}

	value := 42.5
	if err := svc.SimulateTelemetry(context.Background(), "device-1", "temperature", &value, "C"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mqtt.published) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(mqtt.published))
	}
	msg := mqtt.published[0]
	if msg.topic != "devices/device-1/telemetry" {
		t.Errorf("unexpected topic: %s", msg.topic)
	}
	var p telemetryPayload
	if err := json.Unmarshal(msg.payload, &p); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if p.Metric != "temperature" || p.Value != 42.5 || p.Unit != "C" {
		t.Errorf("unexpected payload: %+v", p)
	}
}

func TestSimulateTelemetryRandomizesValueWhenOmitted(t *testing.T) {
	mqtt := &fakeMQTT{}
	svc := &IoTService{mqtt: mqtt, log: zap.NewNop()}

	if err := svc.SimulateTelemetry(context.Background(), "device-1", "humidity", nil, "%"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var p telemetryPayload
	if err := json.Unmarshal(mqtt.published[0].payload, &p); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if p.Value < 30 || p.Value > 90 {
		t.Errorf("expected a plausible humidity value (30-90), got %v", p.Value)
	}
}

func TestSimulateTelemetryReturnsErrWhenMQTTNotConfigured(t *testing.T) {
	svc := &IoTService{mqtt: nil, log: zap.NewNop()}

	err := svc.SimulateTelemetry(context.Background(), "device-1", "temperature", nil, "")
	if err == nil {
		t.Fatal("expected an error when mqtt is nil")
	}
}

func TestSendCommandPublishesToCommandTopic(t *testing.T) {
	mqtt := &fakeMQTT{}
	svc := &IoTService{mqtt: mqtt, log: zap.NewNop()}

	if err := svc.SendCommand(context.Background(), "device-1", "reboot", map[string]any{"delaySec": float64(5)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mqtt.published) != 1 {
		t.Fatalf("expected 1 publish, got %d", len(mqtt.published))
	}
	msg := mqtt.published[0]
	if msg.topic != "devices/device-1/commands" {
		t.Errorf("unexpected topic: %s", msg.topic)
	}
	var body map[string]any
	if err := json.Unmarshal(msg.payload, &body); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if body["command"] != "reboot" {
		t.Errorf("unexpected command in payload: %+v", body)
	}
}

func TestSendCommandReturnsErrWhenMQTTNotConfigured(t *testing.T) {
	svc := &IoTService{mqtt: nil, log: zap.NewNop()}

	err := svc.SendCommand(context.Background(), "device-1", "reboot", nil)
	if err == nil {
		t.Fatal("expected an error when mqtt is nil")
	}
}

func TestSendCommandPropagatesPublishError(t *testing.T) {
	mqtt := &fakeMQTT{pubErr: errors.New("broker unreachable")}
	svc := &IoTService{mqtt: mqtt, log: zap.NewNop()}

	err := svc.SendCommand(context.Background(), "device-1", "reboot", nil)
	if err == nil {
		t.Fatal("expected the publish error to propagate")
	}
}

func TestDeviceIDFromTopic(t *testing.T) {
	cases := map[string]string{
		"devices/abc-123/telemetry": "abc-123",
		"devices/abc/commands":      "", // wrong suffix
		"devices/telemetry":         "", // too few segments
		"garbage":                   "",
	}
	for topic, want := range cases {
		if got := deviceIDFromTopic(topic); got != want {
			t.Errorf("deviceIDFromTopic(%q) = %q, want %q", topic, got, want)
		}
	}
}

func TestRandomValueForRanges(t *testing.T) {
	cases := map[string][2]float64{
		"temperature": {15, 35},
		"humidity":    {30, 90},
		"battery":     {0, 100},
		"unknown":     {0, 100},
	}
	for metric, r := range cases {
		v := randomValueFor(metric)
		if v < r[0] || v > r[1] {
			t.Errorf("randomValueFor(%q) = %v, want in range [%v,%v]", metric, v, r[0], r[1])
		}
	}
}
