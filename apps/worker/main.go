package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	activitySvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/activity/service"
	notifSvc    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	pkglogger   "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/logger"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/worker"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/worker/tasks"
	"go.uber.org/zap"
)

var Version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: config: %v\n", err)
		os.Exit(1)
	}

	zapLog, err := pkglogger.New(cfg.App.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker: logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLog.Sync() //nolint:errcheck

	mongo, err := database.NewMongoDB(cfg.Mongo)
	if err != nil {
		zapLog.Fatal("worker: mongo", zap.Error(err))
	}
	defer mongo.Disconnect(context.Background())

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		zapLog.Fatal("worker: redis", zap.Error(err))
	}
	defer rdb.Close()

	// Services
	actSvc := activitySvc.New(mongo)
	_ = actSvc

	mailer := email.NewMailerFromConfig(cfg, zapLog)

	var fcmSender *notifSvc.FCMSender
	if cfg.Firebase.CredentialsFile != "" {
		fcmSender, err = notifSvc.NewFCMSender(cfg.Firebase.CredentialsFile)
		if err != nil {
			zapLog.Warn("worker: FCM init failed", zap.Error(err))
		}
	}

	// promoQueue is nil here: a worker is the terminal "do the actual send"
	// layer, so Broadcast should run its synchronous per-user path directly
	// rather than re-enqueue — only the API (apps/api/app/app.go) routes
	// broadcast email through the promotional queue.
	notifService := notifSvc.New(mongo, fcmSender, mailer, nil, zapLog)

	// Worker server
	redisAddr := fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port)
	srv := worker.NewServer(redisAddr, cfg.Redis.Password, cfg.Redis.DB, cfg.Worker.Concurrency)

	emailHandler   := tasks.NewEmailTaskHandler(mailer, zapLog)
	notifHandler   := tasks.NewNotificationTaskHandler(notifService, zapLog)
	sessionHandler := tasks.NewSessionCleanupHandler(zapLog)

	tasks.RegisterAll(srv, emailHandler, sessionHandler)
	srv.Handle(tasks.TaskSendNotification, notifHandler.HandleSend)
	srv.Handle(tasks.TaskBroadcast,        notifHandler.HandleBroadcast)
	srv.Handle(tasks.TaskCleanStaleTokens, notifHandler.HandleCleanStaleTokens)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		zapLog.Info("worker starting", zap.Int("concurrency", cfg.Worker.Concurrency))
		if err := srv.Start(); err != nil {
			zapLog.Fatal("worker start", zap.Error(err))
		}
	}()

	<-quit
	zapLog.Info("worker shutting down")
	srv.Shutdown()
	zapLog.Info("worker stopped")
}
