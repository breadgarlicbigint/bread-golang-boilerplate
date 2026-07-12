// Command worker-kafka is the Kafka background worker — the Kafka parallel of
// apps/worker (Redis/Asynq). It consumes jobs from the Kafka topic and dispatches
// them through the transport-agnostic handlers in pkg/queue/tasks.
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
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/kafka"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/tasks"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"go.uber.org/zap"
)

var Version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker-kafka: config: %v\n", err)
		os.Exit(1)
	}

	zapLog, err := pkglogger.New(cfg.App.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker-kafka: logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync() //nolint:errcheck

	mongo, err := database.NewMongoDB(cfg.Mongo)
	if err != nil {
		zapLog.Fatal("worker-kafka: mongo", zap.Error(err))
	}
	defer mongo.Disconnect(context.Background())

	// ── Services ────────────────────────────────────────────────────────────────
	mailer := email.NewMailerFromConfig(cfg, zapLog)

	var fcmSender *notifSvc.FCMSender
	if cfg.Firebase.CredentialsFile != "" {
		fcmSender, err = notifSvc.NewFCMSender(cfg.Firebase.CredentialsFile)
		if err != nil {
			zapLog.Warn("worker-kafka: FCM init failed", zap.Error(err))
		}
	}
	notifService := notifSvc.New(mongo, fcmSender, mailer, zapLog)

	// ── Consumer ────────────────────────────────────────────────────────────────
	consumer, err := kafka.NewConsumer(cfg.Kafka, zapLog)
	if err != nil {
		zapLog.Fatal("worker-kafka: connect", zap.Error(err))
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

	zapLog.Info("worker-kafka starting", zap.String("version", Version))
	if err := consumer.Start(ctx); err != nil {
		zapLog.Fatal("worker-kafka: run", zap.Error(err))
	}
	zapLog.Info("worker-kafka stopped")
}
