// Package mqtt is a thin wrapper around eclipse/paho.mqtt.golang — connect,
// publish, subscribe, close. It has zero domain knowledge; modules/iot is
// the one caller today, publishing/subscribing to device topics.
package mqtt

import (
	"fmt"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Config configures the broker connection. BrokerURL empty means MQTT is
// disabled — callers follow the same "nil = skip silently" convention as
// email.NewMailerFromConfig / Firebase FCM elsewhere in this codebase.
type Config struct {
	BrokerURL string // e.g. tcp://mosquitto:1883
	ClientID  string
	Username  string
	Password  string
}

// MessageHandler receives an incoming message's topic and raw payload.
type MessageHandler func(topic string, payload []byte)

// Client wraps a single paho connection.
type Client struct {
	cli paho.Client
}

// New connects to the broker and blocks (up to a 5s timeout) until the
// connection succeeds or fails — callers get a definite up/down answer at
// startup rather than a client that silently retries forever in the
// background before anything has ever worked.
func New(cfg Config) (*Client, error) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectTimeout(5 * time.Second)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	cli := paho.NewClient(opts)
	token := cli.Connect()
	if !token.WaitTimeout(5 * time.Second) {
		return nil, fmt.Errorf("mqtt: connect timeout dialing %s", cfg.BrokerURL)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt: connect: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Publish sends payload to topic at QoS 1 (at-least-once) and waits for
// broker acknowledgment.
func (c *Client) Publish(topic string, payload []byte) error {
	token := c.cli.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

// Subscribe registers handler for every message received on topic
// (supports MQTT wildcards, e.g. "devices/+/telemetry"). handler runs on
// paho's internal goroutine — do the real work (Mongo write, hub publish)
// quickly or hand off, don't block it indefinitely.
func (c *Client) Subscribe(topic string, handler MessageHandler) error {
	token := c.cli.Subscribe(topic, 1, func(_ paho.Client, msg paho.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	token.Wait()
	return token.Error()
}

// Close disconnects, waiting up to 250ms for in-flight work to settle.
func (c *Client) Close() {
	c.cli.Disconnect(250)
}
