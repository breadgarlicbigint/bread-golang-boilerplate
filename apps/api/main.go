package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/apps/api/app"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/logger"
	"go.uber.org/zap"
)

// Version is set at build time via -ldflags "-X main.Version=x.y.z"
var Version = "dev"

// @title           Bread Golang Boilerplate
// @version         1.0
// @description     Production-ready Go REST API — JWT, RBAC, MongoDB, Redis, WebAuthn, FCM.
// @termsOfService  http://swagger.io/terms/
// @contact.name    API Support
// @contact.email   support@example.com
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @host            localhost:3000
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("config: " + err.Error())
	}
	cfg.App.Version = Version

	log, err := logger.New(cfg.App.Env)
	if err != nil {
		panic("logger: " + err.Error())
	}
	defer log.Sync() //nolint:errcheck

	mongo, err := database.NewMongoDB(cfg.Mongo)
	if err != nil {
		log.Fatal("mongo connect", zap.Error(err))
	}
	log.Info("mongo connected", zap.String("db", cfg.Mongo.DBName))

	rdb, err := database.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatal("redis connect", zap.Error(err))
	}
	log.Info("redis connected")

	application, err := app.New(cfg, log, mongo, rdb)
	if err != nil {
		log.Fatal("app init", zap.Error(err))
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := application.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server start", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		log.Error("shutdown error", zap.Error(err))
		os.Exit(1)
	}
	log.Info("server stopped gracefully")
}
