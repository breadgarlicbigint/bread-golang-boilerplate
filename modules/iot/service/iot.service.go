package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	iot "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/iot"
	iotEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/iot/entity"
	mqttpkg "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/mqtt"
	realtime "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
)

const telemetryCol = "device_telemetry"

// telemetryTopicFmt / commandTopicFmt / telemetryWildcard are the MQTT
// topic conventions this module owns. A real device would publish to
// telemetryTopicFmt and subscribe to commandTopicFmt; here the API plays
// both device (via SimulateTelemetry) and server.
const (
	telemetryTopicFmt = "devices/%s/telemetry"
	commandTopicFmt   = "devices/%s/commands"
	telemetryWildcard = "devices/+/telemetry"
)

// TelemetryTopic is the modules/realtime pub/sub topic persisted readings
// are forwarded to — subscribe a WebSocket/SSE client to it (or watch it on
// the Realtime page in web/) to see MQTT telemetry arrive live.
const TelemetryTopic = "iot:telemetry"

// Publisher is the minimal MQTT capability SimulateTelemetry/SendCommand
// need — satisfied by *pkg/mqtt.Client. Kept narrow so unit tests can fake
// it without a real broker.
type Publisher interface {
	Publish(topic string, payload []byte) error
}

// Subscriber is the minimal MQTT capability Start needs. Declared with
// mqttpkg.MessageHandler (not an inline func type) because Go interface
// satisfaction requires identical named parameter types — *mqtt.Client's
// own Subscribe method takes exactly this named type.
type Subscriber interface {
	Subscribe(topic string, handler mqttpkg.MessageHandler) error
}

// MQTTClient is everything this service needs from the broker connection.
// *pkg/mqtt.Client satisfies it directly.
type MQTTClient interface {
	Publisher
	Subscriber
}

// RealtimePublisher forwards persisted telemetry onto modules/realtime's
// generic pub/sub topic. Optional — nil just means telemetry is persisted
// but not pushed live.
type RealtimePublisher interface {
	PublishToTopic(topic string, evt realtime.Event) (int, error)
}

type telemetryPayload struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	Unit   string  `json:"unit,omitempty"`
}

// IoTService owns the device_telemetry collection and the MQTT
// subscription that fills it. mqtt is nil when MQTT_BROKER_URL is unset
// (see pkg/mqtt.Config) — every method treats that as "MQTT disabled",
// same convention as a nil *email.Mailer or *FCMSender elsewhere.
type IoTService struct {
	col      *mongo.Collection
	mqtt     MQTTClient
	realtime RealtimePublisher
	log      *zap.Logger
}

func New(db *database.MongoDB, mqttClient MQTTClient, realtimePub RealtimePublisher, log *zap.Logger) *IoTService {
	return &IoTService{
		col:      db.Collection(telemetryCol),
		mqtt:     mqttClient,
		realtime: realtimePub,
		log:      log,
	}
}

func (s *IoTService) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "deviceId", Value: 1}, {Key: "recordedAt", Value: -1}}, Options: options.Index().SetName("idx_device_recorded")},
		// TTL: auto-delete telemetry older than 1 day — see entity doc comment.
		{Keys: bson.D{{Key: "recordedAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(86400).SetName("idx_ttl_1d")},
	})
	return err
}

// Start subscribes to every device's telemetry topic. Call once at startup
// after New; a nil mqtt client makes this a no-op (MQTT not configured).
func (s *IoTService) Start() error {
	if s.mqtt == nil {
		return nil
	}
	return s.mqtt.Subscribe(telemetryWildcard, s.handleTelemetry)
}

// ── Publish side ─────────────────────────────────────────────────────────────

func (s *IoTService) SimulateTelemetry(_ context.Context, deviceID, metric string, value *float64, unit string) error {
	if s.mqtt == nil {
		return errors.ErrMQTTNotConfigured
	}
	v := 0.0
	if value != nil {
		v = *value
	} else {
		v = randomValueFor(metric)
	}
	payload, err := json.Marshal(telemetryPayload{Metric: metric, Value: v, Unit: unit})
	if err != nil {
		return err
	}
	return s.mqtt.Publish(fmt.Sprintf(telemetryTopicFmt, deviceID), payload)
}

func (s *IoTService) SendCommand(_ context.Context, deviceID, command string, data map[string]any) error {
	if s.mqtt == nil {
		return errors.ErrMQTTNotConfigured
	}
	payload, err := json.Marshal(map[string]any{"command": command, "data": data})
	if err != nil {
		return err
	}
	return s.mqtt.Publish(fmt.Sprintf(commandTopicFmt, deviceID), payload)
}

// randomValueFor picks a plausible reading so a one-click "Simulate" button
// in the web test client produces sensible-looking data without the caller
// having to supply a value.
func randomValueFor(metric string) float64 {
	switch strings.ToLower(metric) {
	case "temperature":
		return 15 + rand.Float64()*20 // 15–35 °C
	case "humidity":
		return 30 + rand.Float64()*60 // 30–90 %
	case "battery":
		return rand.Float64() * 100 // 0–100 %
	default:
		return rand.Float64() * 100
	}
}

// ── Subscribe side ────────────────────────────────────────────────────────────

// handleTelemetry runs on paho's internal goroutine for every message on
// telemetryWildcard — persist, then best-effort forward to the realtime hub.
func (s *IoTService) handleTelemetry(topic string, payload []byte) {
	deviceID := deviceIDFromTopic(topic)
	if deviceID == "" {
		return
	}
	var p telemetryPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		s.log.Warn("iot: malformed telemetry payload", zap.String("topic", topic), zap.Error(err))
		return
	}

	reading := &iotEntity.DeviceTelemetry{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		Metric:     p.Metric,
		Value:      p.Value,
		Unit:       p.Unit,
		RecordedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.col.InsertOne(ctx, reading); err != nil {
		s.log.Warn("iot: failed to persist telemetry", zap.String("deviceId", deviceID), zap.Error(err))
		return
	}

	if s.realtime != nil {
		_, _ = s.realtime.PublishToTopic(TelemetryTopic, realtime.Event{
			Type:  "iot.telemetry",
			Title: deviceID,
			Body:  fmt.Sprintf("%s = %v %s", p.Metric, p.Value, p.Unit),
			Data: map[string]any{
				"deviceId": deviceID,
				"metric":   p.Metric,
				"value":    p.Value,
				"unit":     p.Unit,
			},
		})
	}
}

// deviceIDFromTopic extracts "abc123" from "devices/abc123/telemetry".
func deviceIDFromTopic(topic string) string {
	parts := strings.Split(topic, "/")
	if len(parts) != 3 || parts[0] != "devices" || parts[2] != "telemetry" {
		return ""
	}
	return parts[1]
}

// ── Query side ─────────────────────────────────────────────────────────────────

// ListTelemetry returns paginated readings, most recent first, optionally
// filtered to a single device.
func (s *IoTService) ListTelemetry(ctx context.Context, deviceID string, page, perPage int) ([]*iotEntity.DeviceTelemetry, int64, error) {
	filter := bson.M{}
	if deviceID != "" {
		filter["deviceId"] = deviceID
	}
	total, _ := s.col.CountDocuments(ctx, filter)
	opts := options.Find().
		SetSort(bson.D{{Key: "recordedAt", Value: -1}}).
		SetSkip(int64((page - 1) * perPage)).
		SetLimit(int64(perPage))

	cur, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var readings []*iotEntity.DeviceTelemetry
	return readings, total, cur.All(ctx, &readings)
}

var _ iot.Service = (*IoTService)(nil)
