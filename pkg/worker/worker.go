package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/hibiken/asynq"
)

// Task type constants — use these as the task "type" string. Kept identical
// to pkg/queue's constants so payloads are interchangeable across transports.
const (
	TaskSendEmail              = "email:send"
	TaskSendPromotionalEmail   = "email:send:promotional"
	TaskSendPushNotif          = "notification:push"
	TaskProcessUpload          = "upload:process"
	TaskCleanupExpiredSessions = "session:cleanup"
)

// ── Payload types ─────────────────────────────────────────────────────────────

type SendEmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`           // HTML body
	Text    string `json:"text,omitempty"` // optional plain-text alternative
}

type PushNotifPayload struct {
	UserID  string `json:"userId"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Data    map[string]string `json:"data,omitempty"`
}

// ── Client (enqueuer) ────────────────────────────────────────────────────────

type Client struct {
	client *asynq.Client
}

func NewClient(redisAddr, redisPassword string, db int) *Client {
	c := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       db,
	})
	return &Client{client: c}
}

func (c *Client) Enqueue(taskType string, payload interface{}, opts ...asynq.Option) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("worker: marshal payload: %w", err)
	}
	task := asynq.NewTask(taskType, b, opts...)
	_, err = c.client.Enqueue(task)
	return err
}

// EnqueueEmail queues a pre-rendered email for the background worker to deliver.
// The caller renders the subject/HTML/text (e.g. via email.LocalizedMailer) so
// the worker only performs the retryable network send. Failed sends are retried
// by asynq up to MaxRetry times on the "default" queue.
//
// Defining this convenience method here lets callers depend on a small
// domain-specific interface (EnqueueEmail) instead of importing asynq to
// construct task types and options themselves.
func (c *Client) EnqueueEmail(to, subject, html, text string) error {
	return c.Enqueue(
		TaskSendEmail,
		SendEmailPayload{To: to, Subject: subject, Body: html, Text: text},
		asynq.MaxRetry(5),
		asynq.Queue("default"),
	)
}

// EnqueuePromotionalEmail queues a bulk/marketing email on the "low" priority
// queue with fewer retries than EnqueueEmail — promotional traffic favors
// throughput over guaranteed delivery, so it shouldn't compete with (or retry
// as hard as) transactional email on the same broker. See pkg/queue/router
// for routing this task type to an entirely different backend instead.
func (c *Client) EnqueuePromotionalEmail(to, subject, html, text string) error {
	return c.Enqueue(
		TaskSendPromotionalEmail,
		SendEmailPayload{To: to, Subject: subject, Body: html, Text: text},
		asynq.MaxRetry(2),
		asynq.Queue("low"),
	)
}

func (c *Client) Close() error { return c.client.Close() }

// ── Server (processor) ────────────────────────────────────────────────────────

type HandlerFunc func(ctx context.Context, t *asynq.Task) error

type Server struct {
	server  *asynq.Server
	mux     *asynq.ServeMux
}

func NewServer(redisAddr, redisPassword string, db, concurrency int) *Server {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr, Password: redisPassword, DB: db},
		asynq.Config{
			Concurrency: concurrency,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("worker: task %s failed: %v", task.Type(), err)
			}),
		},
	)
	return &Server{server: srv, mux: asynq.NewServeMux()}
}

// Handle registers a handler for a task type.
func (s *Server) Handle(taskType string, fn HandlerFunc) {
	s.mux.HandleFunc(taskType, fn)
}

// Start begins processing tasks (blocking).
func (s *Server) Start() error {
	return s.server.Run(s.mux)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() { s.server.Shutdown() }
