// Command worker-rabbitmq is the RabbitMQ background worker — the AMQP parallel
// of apps/worker (Redis/Asynq). It consumes jobs from the RabbitMQ exchange and
// dispatches them through the transport-agnostic handlers in pkg/queue/tasks.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	notifSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	pkglogger "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/logger"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/rabbitmq"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/tasks"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"go.uber.org/zap"
)

var Version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker-rabbitmq: config: %v\n", err)
		os.Exit(1)
	}

	zapLog, err := pkglogger.New(cfg.App.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker-rabbitmq: logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync() //nolint:errcheck

	mongo, err := database.NewMongoDB(cfg.Mongo)
	if err != nil {
		zapLog.Fatal("worker-rabbitmq: mongo", zap.Error(err))
	}
	defer mongo.Disconnect(context.Background())

	// ── Services ────────────────────────────────────────────────────────────────
	mailer := email.NewMailerFromConfig(cfg, zapLog)

	var fcmSender *notifSvc.FCMSender
	if cfg.Firebase.CredentialsFile != "" {
		fcmSender, err = notifSvc.NewFCMSender(cfg.Firebase.CredentialsFile)
		if err != nil {
			zapLog.Warn("worker-rabbitmq: FCM init failed", zap.Error(err))
		}
	}
	notifService := notifSvc.New(mongo, fcmSender, mailer, zapLog)

	// ── Consumer ────────────────────────────────────────────────────────────────
	consumer, err := rabbitmq.NewConsumer(cfg.RabbitMQ, zapLog)
	if err != nil {
		zapLog.Fatal("worker-rabbitmq: connect", zap.Error(err))
	}
	defer consumer.Close()

	tasks.RegisterAll(consumer,
		tasks.NewEmailTaskHandler(mailer, zapLog),
		tasks.NewSessionCleanupHandler(zapLog),
	)
	tasks.RegisterNotifications(consumer, tasks.NewNotificationTaskHandler(notifService, zapLog))

	// ── Run until signal ────────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	zapLog.Info("worker-rabbitmq starting", zap.String("version", Version))
	if err := consumer.Start(ctx); err != nil {
		zapLog.Fatal("worker-rabbitmq: run", zap.Error(err))
	}
	zapLog.Info("worker-rabbitmq stopped")
}
