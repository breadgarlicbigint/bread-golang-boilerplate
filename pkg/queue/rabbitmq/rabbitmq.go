// Package rabbitmq implements the queue.Publisher and queue.Consumer contract
// over RabbitMQ (AMQP 0-9-1) using rabbitmq/amqp091-go.
//
// Topology: a single durable direct exchange (default "bread.tasks"). Jobs are
// published with the task-type string as the routing key. A worker declares one
// durable queue (default "bread.worker") and binds it to the exchange for every
// task type it has a handler for, so only relevant jobs are delivered. Messages
// are persistent and acknowledged manually — Ack on success, Nack (no requeue)
// on handler error so a poison message is dead-lettered rather than looped.
package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

// ── Publisher ─────────────────────────────────────────────────────────────────

// Publisher enqueues jobs onto a RabbitMQ exchange. Safe for concurrent use.
type Publisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
	mu       sync.Mutex // amqp channels are not safe for concurrent publish
}

// NewPublisher dials RabbitMQ and declares the durable direct exchange.
func NewPublisher(cfg config.RabbitMQConfig) (*Publisher, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq: channel: %w", err)
	}
	if err := declareExchange(ch, cfg.Exchange); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Publisher{conn: conn, ch: ch, exchange: cfg.Exchange}, nil
}

// Enqueue marshals payload to JSON and publishes it under taskType (routing key).
func (p *Publisher) Enqueue(taskType string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("rabbitmq: marshal payload: %w", err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ch.PublishWithContext(
		context.Background(),
		p.exchange, // exchange
		taskType,   // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Type:         taskType,
			Body:         b,
		},
	)
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

// Close tears down the channel and connection.
func (p *Publisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

// ── Consumer ──────────────────────────────────────────────────────────────────

// Consumer runs a RabbitMQ worker: register handlers, then Start.
type Consumer struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
	queue    string
	prefetch int
	log      *zap.Logger
	handlers map[string]queue.HandlerFunc
}

// NewConsumer dials RabbitMQ and declares the exchange + durable work queue.
func NewConsumer(cfg config.RabbitMQConfig, log *zap.Logger) (*Consumer, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq: channel: %w", err)
	}
	if err := declareExchange(ch, cfg.Exchange); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if _, err := ch.QueueDeclare(cfg.Queue, true, false, false, false, nil); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq: queue declare: %w", err)
	}
	prefetch := cfg.Prefetch
	if prefetch <= 0 {
		prefetch = 10
	}
	return &Consumer{
		conn: conn, ch: ch,
		exchange: cfg.Exchange, queue: cfg.Queue, prefetch: prefetch,
		log:      log,
		handlers: make(map[string]queue.HandlerFunc),
	}, nil
}

// Handle registers fn for a task type. Call before Start.
func (c *Consumer) Handle(taskType string, fn queue.HandlerFunc) {
	c.handlers[taskType] = fn
}

// Start binds the work queue to every registered task type and consumes until
// ctx is canceled. Blocks; returns nil on clean shutdown.
func (c *Consumer) Start(ctx context.Context) error {
	// Bind the queue for each registered task type (routing key).
	for taskType := range c.handlers {
		if err := c.ch.QueueBind(c.queue, taskType, c.exchange, false, nil); err != nil {
			return fmt.Errorf("rabbitmq: bind %q: %w", taskType, err)
		}
	}
	if err := c.ch.Qos(c.prefetch, 0, false); err != nil {
		return fmt.Errorf("rabbitmq: qos: %w", err)
	}

	deliveries, err := c.ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq: consume: %w", err)
	}

	c.log.Info("rabbitmq worker consuming",
		zap.String("queue", c.queue), zap.Int("prefetch", c.prefetch), zap.Int("handlers", len(c.handlers)))

	// A pool of workers processes deliveries concurrently up to the prefetch.
	var wg sync.WaitGroup
	for i := 0; i < c.prefetch; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for d := range deliveries {
				c.dispatch(ctx, d)
			}
		}()
	}

	<-ctx.Done()
	// Cancel the consumer so the deliveries channel closes and workers drain.
	_ = c.ch.Cancel("", false)
	wg.Wait()
	c.log.Info("rabbitmq worker stopped")
	return nil
}

// dispatch routes one delivery to its handler and acks/nacks accordingly.
func (c *Consumer) dispatch(ctx context.Context, d amqp.Delivery) {
	taskType := d.Type
	if taskType == "" {
		taskType = d.RoutingKey
	}
	h, ok := c.handlers[taskType]
	if !ok {
		c.log.Warn("rabbitmq: no handler for task", zap.String("type", taskType))
		_ = d.Nack(false, false)
		return
	}
	if err := h(ctx, queue.Delivery{Type: taskType, Payload: d.Body}); err != nil {
		c.log.Error("rabbitmq: task failed", zap.String("type", taskType), zap.Error(err))
		_ = d.Nack(false, false) // dead-letter rather than requeue-loop
		return
	}
	_ = d.Ack(false)
}

// Close tears down the channel and connection.
func (c *Consumer) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// declareExchange declares the shared durable direct exchange (idempotent).
func declareExchange(ch *amqp.Channel, exchange string) error {
	if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("rabbitmq: exchange declare: %w", err)
	}
	return nil
}

// Compile-time checks that the RabbitMQ types satisfy the queue contract.
var (
	_ queue.Publisher = (*Publisher)(nil)
	_ queue.Consumer  = (*Consumer)(nil)
)
