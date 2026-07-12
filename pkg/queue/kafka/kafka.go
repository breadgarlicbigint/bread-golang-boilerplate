// Package kafka implements the queue.Publisher and queue.Consumer contract over
// Apache Kafka using the pure-Go segmentio/kafka-go client (no CGO / librdkafka,
// so it builds with CGO_ENABLED=0 like the rest of the project).
//
// Topology: a single topic (default "bread.tasks") carries every job. The
// task-type string is the Kafka message key, so the consumer dispatches by key
// and same-keyed jobs preserve per-key ordering. The worker joins a consumer
// group (default "bread.worker"); scale throughput by running more worker
// instances (Kafka assigns partitions across the group). Offsets are committed
// after a message is processed (at-least-once).
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// splitBrokers turns "a:9092,b:9092" into a broker slice.
func splitBrokers(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ── Publisher ─────────────────────────────────────────────────────────────────

// Publisher produces jobs to a Kafka topic. *kafkago.Writer is safe for
// concurrent use, so Publisher is too.
type Publisher struct {
	writer *kafkago.Writer
}

// NewPublisher builds a Kafka producer for the configured topic.
func NewPublisher(cfg config.KafkaConfig) (*Publisher, error) {
	brokers := splitBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka: no brokers configured")
	}
	w := &kafkago.Writer{
		Addr:                   kafkago.TCP(brokers...),
		Topic:                  cfg.Topic,
		Balancer:               &kafkago.Hash{}, // route by key → per-task-type ordering
		AllowAutoTopicCreation: true,            // dev convenience; pre-create in prod
	}
	return &Publisher{writer: w}, nil
}

// Enqueue marshals payload to JSON and produces it with taskType as the key.
func (p *Publisher) Enqueue(taskType string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("kafka: marshal payload: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(taskType),
		Value: b,
	})
}

// EnqueueEmail queues a pre-rendered email (queue.TaskSendEmail). Mirrors
// worker.Client.EnqueueEmail so any backend can back the welcome-email flow.
func (p *Publisher) EnqueueEmail(to, subject, html, text string) error {
	return p.Enqueue(queue.TaskSendEmail, queue.SendEmailPayload{
		To: to, Subject: subject, Body: html, Text: text,
	})
}

// EnqueuePromotionalEmail queues a pre-rendered bulk/marketing email
// (queue.TaskSendPromotionalEmail) — same payload shape as EnqueueEmail,
// different task type so pkg/queue/router can send it to a different backend.
func (p *Publisher) EnqueuePromotionalEmail(to, subject, html, text string) error {
	return p.Enqueue(queue.TaskSendPromotionalEmail, queue.SendEmailPayload{
		To: to, Subject: subject, Body: html, Text: text,
	})
}

// Close flushes and closes the writer.
func (p *Publisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}

// ── Consumer ──────────────────────────────────────────────────────────────────

// Consumer runs a Kafka worker: register handlers, then Start.
type Consumer struct {
	reader   *kafkago.Reader
	log      *zap.Logger
	handlers map[string]queue.HandlerFunc
}

// NewConsumer builds a consumer-group reader for the configured topic.
func NewConsumer(cfg config.KafkaConfig, log *zap.Logger) (*Consumer, error) {
	brokers := splitBrokers(cfg.Brokers)
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka: no brokers configured")
	}
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:  brokers,
		Topic:    cfg.Topic,
		GroupID:  cfg.GroupID, // consumer group → partitions shared across instances
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})
	return &Consumer{
		reader:   r,
		log:      log,
		handlers: make(map[string]queue.HandlerFunc),
	}, nil
}

// Handle registers fn for a task type. Call before Start.
func (c *Consumer) Handle(taskType string, fn queue.HandlerFunc) {
	c.handlers[taskType] = fn
}

// Start consumes messages until ctx is canceled. Blocks; returns nil on clean
// shutdown. Uses fetch-then-commit for at-least-once delivery.
func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("kafka worker consuming",
		zap.String("topic", c.reader.Config().Topic),
		zap.String("group", c.reader.Config().GroupID),
		zap.Int("handlers", len(c.handlers)))

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			// ctx canceled → clean shutdown.
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				c.log.Info("kafka worker stopped")
				return nil
			}
			return fmt.Errorf("kafka: fetch: %w", err)
		}
		c.dispatch(ctx, m)
		// Commit after processing (at-least-once). A failed job is logged and
		// committed rather than looped forever — hook a DLQ/retry policy here.
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			c.log.Error("kafka: commit failed", zap.Error(err))
		}
	}
}

// dispatch routes one message to its handler, keyed by the message key.
func (c *Consumer) dispatch(ctx context.Context, m kafkago.Message) {
	taskType := string(m.Key)
	h, ok := c.handlers[taskType]
	if !ok {
		c.log.Warn("kafka: no handler for task", zap.String("type", taskType))
		return
	}
	if err := h(ctx, queue.Delivery{Type: taskType, Payload: m.Value}); err != nil {
		c.log.Error("kafka: task failed", zap.String("type", taskType), zap.Error(err))
	}
}

// Close closes the reader.
func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// Compile-time checks that the Kafka types satisfy the queue contract.
var (
	_ queue.Publisher = (*Publisher)(nil)
	_ queue.Consumer  = (*Consumer)(nil)
)
