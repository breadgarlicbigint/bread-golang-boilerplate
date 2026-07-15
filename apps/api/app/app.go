package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/getsentry/sentry-go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	activitySvc  "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/activity/service"
	analyticsHdl "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/analytics/handler"
	analyticsSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/analytics/service"
	apikeySvc    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/apikey/service"
	authHdl      "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/handler"
	authSvc      "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/service"
	appverHdl    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/handler"
	appverMw     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/middleware"
	appverSvc    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
	iotHdl       "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/iot/handler"
	iotSvc       "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/iot/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	healthHdl    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/health/handler"
	mobileHdl    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/mobile/handler"
	mobileSvc    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/mobile/service"
	notifHdl     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/handler"
	notifSvc     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/service"
	passkeyHdl   "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/handler"
	passkeyRepo  "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/repository"
	passkeySvc   "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/service"
	realtimeHdl  "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime/handler"
	realtimeSvc  "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime/service"
	tenantMw     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/middleware"
	tenantSvc    "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/service"
	userHdl      "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/handler"
	userRepo     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/repository"
	roleHdl      "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/handler"
	roleRepo     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/repository"
	roleSvc      "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/service"
	userSvc      "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/service"
	userentity   "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	pkgi18n      "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/i18n"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/metrics"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/mqtt"
	pkgrealtime  "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/realtime"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger   "github.com/swaggo/gin-swagger"
	jwtpkg       "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/jwt"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/sms"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/worker"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/kafka"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/rabbitmq"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue/router"
)

// App holds all wired-up dependencies and the HTTP server.
type App struct {
	cfg        *config.Config
	log        *zap.Logger
	mongo      *database.MongoDB
	rdb        *redis.Client
	engine     *gin.Engine
	server     *http.Server
	jobQueue   queue.Publisher
	mqttClient *mqtt.Client
}

// New wires all modules and returns a ready-to-start App.
func New(cfg *config.Config, log *zap.Logger, mongo *database.MongoDB, rdb *redis.Client) (*App, error) {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Every error response (shared/response.Error and friends) and every
	// handleError fallback (shared/response.LogInternal) prints through this
	// logger, so API/application errors always show up on the console.
	response.SetLogger(log)

	// ── Sentry ────────────────────────────────────────────────────────────────
	if cfg.Sentry.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.Sentry.DSN,
			TracesSampleRate: cfg.Sentry.TracesSampleRate,
			Environment:      cfg.App.Env,
		}); err != nil {
			log.Warn("sentry init failed", zap.Error(err))
		}
	}

	// ── i18n ──────────────────────────────────────────────────────────────────
	localesDir := cfg.I18n.LocalesDir
	if localesDir == "" {
		localesDir = "./locales"
	}
	translator, err := pkgi18n.New(localesDir)
	if err != nil {
		log.Warn("i18n: locales not loaded — key passthrough mode", zap.Error(err))
		translator = nil
	}

	// ── Shared packages ───────────────────────────────────────────────────────
	hasher := hash.New(12)

	jwtMgr, err := jwtpkg.New(
		cfg.JWT.AccessPrivateKeyPath, cfg.JWT.AccessPublicKeyPath,
		cfg.JWT.RefreshPrivateKeyPath, cfg.JWT.RefreshPublicKeyPath,
		cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire,
	)
	if err != nil {
		return nil, fmt.Errorf("app: jwt: %w", err)
	}

	// ── Email — MAIL_DRIVER selects SES (default) or SMTP ──────────────────────
	mailer := email.NewMailerFromConfig(cfg, log)

	// ── Localized mailer ─────────────────────────────────────────────────────────
	var localMailer *email.LocalizedMailer
	if mailer != nil && translator != nil {
		localMailer = email.NewLocalizedMailer(mailer, translator)
	}

	// ── Background job queue (QUEUE_DRIVER: redis | rabbitmq | kafka) ───────────
	// The API only enqueues jobs; a separate worker process consumes them.
	// Emails are rendered here (fast) and their retryable delivery is handed off
	// via jobQueue.EnqueueEmail. The publisher MUST match the running worker —
	// QUEUE_DRIVER selects it. See apps/worker, apps/worker-rabbitmq, apps/worker-kafka.
	jobQueue := buildJobQueue(cfg, log)

	// ── Core modules ──────────────────────────────────────────────────────────
	uRepo       := userRepo.New(mongo)
	rRepo       := roleRepo.New(mongo)
	rSvc        := roleSvc.New(rRepo)
	uSvc        := userSvc.New(uRepo, hasher, cfg.Auth, localMailer, jobQueue, cfg.App.URL, log)
	apiKeySvc   := apikeySvc.New(mongo, hasher, cfg.APIKey.Prefix)
	actSvc      := activitySvc.New(mongo)
	authService := authSvc.New(uRepo, uSvc, rRepo, jwtMgr, hasher, rdb, *cfg, log, localMailer)

	// ── Social Auth ───────────────────────────────────────────────────────────
	ghOAuth    := authSvc.NewGitHubOAuth(cfg.GitHub)
	ghHandler  := authHdl.NewGitHub(ghOAuth, uRepo, rdb, authService)

	var appleHandler *authHdl.AppleHandler
	if cfg.Apple.ClientID != "" {
		appleSvc, appleErr := authSvc.NewAppleSignIn(cfg.Apple)
		if appleErr != nil {
			log.Warn("app: Apple Sign In init failed", zap.Error(appleErr))
		} else {
			appleHandler = authHdl.NewApple(appleSvc, uRepo, authService)
		}
	}

	// ── Passkey / Biometrics ──────────────────────────────────────────────────
	pkRepo  := passkeyRepo.New(mongo)
	pkSvc, pkErr := passkeySvc.New(pkRepo, rdb, cfg.WebAuthn.RPID, cfg.WebAuthn.RPOrigin, cfg.WebAuthn.RPName)
	if pkErr != nil {
		log.Warn("app: WebAuthn init failed — passkey routes disabled", zap.Error(pkErr))
		pkSvc = nil
	}

	// ── Multi-tenant ──────────────────────────────────────────────────────────
	tenantService := tenantSvc.New(mongo)

	// ── SMS / WhatsApp ────────────────────────────────────────────────────────
	smsSender     := sms.New(sms.Config{
		AccountSID:   cfg.Twilio.AccountSID,
		AuthToken:    cfg.Twilio.AuthToken,
		FromSMS:      cfg.Twilio.FromSMS,
		FromWhatsApp: cfg.Twilio.FromWhatsApp,
	})
	mobileService := mobileSvc.New(mongo, rdb, smsSender, hasher, actSvc)

	// ── Firebase FCM ──────────────────────────────────────────────────────────
	var fcmSender *notifSvc.FCMSender
	if cfg.Firebase.CredentialsFile != "" {
		fcmSender, err = notifSvc.NewFCMSender(cfg.Firebase.CredentialsFile)
		if err != nil {
			log.Warn("app: Firebase FCM init failed — push disabled", zap.Error(err))
		}
	}
	// ── Realtime (WebSocket / SSE / generic pub-sub) ─────────────────────────────
	realtimeHub := pkgrealtime.NewHub()
	realtimeService := realtimeSvc.New(realtimeHub)

	notifService := notifSvc.New(mongo, fcmSender, mailer, jobQueue, jobQueue, realtimeService, log)

	// ── IoT (MQTT device-telemetry demo) ──────────────────────────────────────
	// Optional, like Firebase/Apple below — MQTT_BROKER_URL empty or a failed
	// dial just disables modules/iot's routes (every call returns
	// errors.ErrMQTTNotConfigured) rather than failing API startup.
	var mqttClient *mqtt.Client
	if cfg.MQTT.BrokerURL != "" {
		mqttClient, err = mqtt.New(mqtt.Config{
			BrokerURL: cfg.MQTT.BrokerURL,
			ClientID:  cfg.MQTT.ClientID,
			Username:  cfg.MQTT.Username,
			Password:  cfg.MQTT.Password,
		})
		if err != nil {
			log.Warn("app: MQTT connect failed — iot routes disabled", zap.Error(err))
			mqttClient = nil
		}
	}
	var iotMQTT iotSvc.MQTTClient
	if mqttClient != nil {
		iotMQTT = mqttClient
	}
	iotService := iotSvc.New(mongo, iotMQTT, realtimeService, log)
	if err := iotService.Start(); err != nil {
		log.Warn("app: MQTT telemetry subscription failed", zap.Error(err))
	}

	// ── App versioning ────────────────────────────────────────────────────────
	versionService := appverSvc.New(mongo)

	// ── Analytics ─────────────────────────────────────────────────────────────
	analyticsService := analyticsSvc.New(mongo)

	// ── Middleware ────────────────────────────────────────────────────────────
	sessionStore := &redisSessionStore{rdb}
	authMw       := middleware.AuthJWTAccess(jwtMgr, sessionStore)
	authWSMw     := middleware.AuthJWTAccessWS(jwtMgr, sessionStore)
	adminMw      := middleware.RoleProtected("admin")
	rateLimiter  := middleware.NewRateLimiter(rdb, cfg.Rate.Requests, cfg.Rate.Period, "global")
	apiKeyMw     := middleware.APIKeyProtected(cfg.APIKey.Header, apiKeySvc)
	versionMw    := appverMw.VersionCheck(versionService)
	activityMw   := middleware.ActivityLogger(actSvc)
	secHeaders   := middleware.SecurityHeaders(middleware.APISecurityConfig())
	_ = apiKeyMw // available for use on specific routes

	// ── Gin engine ────────────────────────────────────────────────────────────
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.Logger(log),
		metrics.GinMiddleware(),
		secHeaders,
		gzip.Gzip(gzip.DefaultCompression),
		cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders: []string{
				"Origin", "Content-Type", "Authorization",
				cfg.APIKey.Header,
				"X-Tenant-ID", "X-App-Version", "X-App-Platform",
				pkgi18n.LangHeader,
			},
			ExposeHeaders:    []string{"X-Request-Id", "X-Version-Status", "X-Current-Version", "X-Cache"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		rateLimiter.Middleware(),
		versionMw,
		activityMw,
	)

	if translator != nil {
		engine.Use(pkgi18n.Middleware(translator))
	}

	// ── Tenant middleware ─────────────────────────────────────────────────────
	var tenantResolver gin.HandlerFunc
	if cfg.Tenant.Enabled {
		tenantResolver = tenantMw.TenantFromHeader(tenantService)
	} else {
		tenantResolver = func(c *gin.Context) { c.Next() }
	}

	// ── Route groups ──────────────────────────────────────────────────────────
	root := engine.Group("")
	healthHdl.New(mongo, rdb, cfg.App.Version).RegisterRoutes(root)
	root.GET("/metrics", metrics.Handler())

	if cfg.App.Env != "production" {
		// Swagger UI — generated by: make swagger
		root.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	v1 := engine.Group("/v1", tenantResolver)

	// Auth (credential + token management)
	authHdl.New(authService).RegisterRoutes(v1, authMw)

	// Social auth
	ghHandler.RegisterRoutes(v1)
	if appleHandler != nil {
		appleHandler.RegisterRoutes(v1)
	}

	// Passkey + Biometrics
	if pkSvc != nil {
		passkeyHdl.New(pkSvc, &userSvcAdapter{uSvc}).RegisterRoutes(v1, authMw)
	}

	// User CRUD
	userHdl.New(uSvc).RegisterRoutes(v1, authMw, adminMw)

	// Roles (used to populate role-selection dropdowns, e.g. admin "create user" form)
	roleHdl.New(rSvc).RegisterRoutes(v1, authMw, adminMw)

	// Mobile verification
	mobileHdl.New(mobileService).RegisterRoutes(v1, authMw)

	// Notifications
	notifHdl.New(notifService).RegisterRoutes(v1, authMw, adminMw)

	// Realtime (WebSocket / SSE / generic pub-sub)
	realtimeHdl.New(realtimeHub, log).RegisterRoutes(v1, authMw, authWSMw, adminMw)

	// IoT (MQTT device-telemetry demo)
	iotHdl.New(iotService).RegisterRoutes(v1, authMw, adminMw)

	// App versioning
	appverHdl.New(versionService).RegisterRoutes(v1, authMw, adminMw)

	// Analytics (admin-only)
	adminV1 := engine.Group("/v1", tenantResolver, authMw, adminMw)
	analyticsHdl.New(analyticsService, rdb, cfg.Auth.MaxPasswordAttempts).RegisterRoutes(adminV1)

	app := &App{
		cfg:        cfg,
		log:        log,
		mongo:      mongo,
		rdb:        rdb,
		engine:     engine,
		jobQueue:   jobQueue,
		mqttClient: mqttClient,
		server: &http.Server{
			Addr:         ":" + cfg.App.Port,
			Handler:      engine,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
	return app, nil
}

// buildPublisher constructs a single queue.Publisher for the given driver
// name. Redis client creation is lazy (never errors); RabbitMQ/Kafka dial
// eagerly — a connection failure is logged and returns nil, letting the
// caller decide whether that's fatal (default driver) or just a fallback
// (a transactional/promotional override that didn't come up).
func buildPublisher(driver string, cfg *config.Config, log *zap.Logger) queue.Publisher {
	switch driver {
	case "rabbitmq":
		pub, err := rabbitmq.NewPublisher(cfg.RabbitMQ)
		if err != nil {
			log.Error("queue: rabbitmq publisher init failed", zap.Error(err))
			return nil
		}
		log.Info("queue: rabbitmq publisher ready", zap.String("exchange", cfg.RabbitMQ.Exchange))
		return pub
	case "kafka":
		pub, err := kafka.NewPublisher(cfg.Kafka)
		if err != nil {
			log.Error("queue: kafka publisher init failed", zap.Error(err))
			return nil
		}
		log.Info("queue: kafka publisher ready", zap.String("topic", cfg.Kafka.Topic))
		return pub
	case "redis", "":
		log.Info("queue: redis/asynq publisher ready")
		return asynqPublisherAdapter{worker.NewClient(
			fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
			cfg.Redis.Password, cfg.Redis.DB,
		)}
	default:
		log.Error("queue: unknown driver", zap.String("driver", driver))
		return nil
	}
}

// buildJobQueue wires the enqueue-side publisher(s) the API uses. Driver
// selects the default backend for every task type; TransactionalDriver and
// PromotionalDriver (both optional, see shared/config QueueConfig) pin
// specific workloads to a different backend — e.g. RabbitMQ for
// reliability-sensitive transactional email (queue.TaskSendEmail), Kafka for
// high-throughput promotional/bulk email (queue.TaskSendPromotionalEmail) —
// via pkg/queue/router. Every driver referenced by any of the three fields
// MUST have its matching worker process running:
//
//	redis (default) → apps/worker           (Redis/Asynq)
//	rabbitmq        → apps/worker-rabbitmq
//	kafka           → apps/worker-kafka
//
// A transactional/promotional override that fails to connect logs a warning
// and falls back to the default driver rather than disabling background jobs
// outright — only a default-driver failure disables jobs entirely (nil),
// matching the nil-mailer convention every caller already treats as "skip
// silently".
//
// NOTE: drivers are read once at startup. Changing them in .env requires a
// full process restart — under `make dev` (air) a .env-only change does NOT
// rebuild (exclude_unchanged), so restart air after switching drivers.
func buildJobQueue(cfg *config.Config, log *zap.Logger) queue.Publisher {
	def := buildPublisher(cfg.Queue.Driver, cfg, log)
	if def == nil {
		log.Error("queue: default driver init failed — background jobs disabled",
			zap.String("driver", cfg.Queue.Driver))
		return nil
	}

	rt := router.New(def)

	if d := cfg.Queue.TransactionalDriver; d != "" && d != cfg.Queue.Driver {
		if pub := buildPublisher(d, cfg, log); pub != nil {
			rt.Route(queue.TaskSendEmail, pub)
			log.Info("queue: routing transactional email", zap.String("driver", d))
		} else {
			log.Warn("queue: transactional driver init failed — transactional email falls back to the default driver",
				zap.String("driver", d))
		}
	}

	if d := cfg.Queue.PromotionalDriver; d != "" && d != cfg.Queue.Driver {
		if pub := buildPublisher(d, cfg, log); pub != nil {
			rt.Route(queue.TaskSendPromotionalEmail, pub)
			log.Info("queue: routing promotional email", zap.String("driver", d))
		} else {
			log.Warn("queue: promotional driver init failed — promotional email falls back to the default driver",
				zap.String("driver", d))
		}
	}

	return rt
}

func (a *App) Start() error {
	a.log.Info("server starting",
		zap.String("addr", a.server.Addr),
		zap.String("env", a.cfg.App.Env),
		zap.String("version", a.cfg.App.Version),
	)
	return a.server.ListenAndServe()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.log.Info("graceful shutdown initiated")
	if err := a.server.Shutdown(ctx); err != nil {
		return err
	}
	_ = a.mongo.Disconnect(ctx)
	if a.jobQueue != nil {
		_ = a.jobQueue.Close()
	}
	if a.mqttClient != nil {
		a.mqttClient.Close()
	}
	_ = a.rdb.Close()
	sentry.Flush(2 * time.Second)
	return nil
}

// ── Adapters ──────────────────────────────────────────────────────────────────

type redisSessionStore struct{ rdb *redis.Client }

func (r *redisSessionStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	n, err := r.rdb.Exists(ctx, "session:"+sessionID).Result()
	return n > 0, err
}

type userSvcAdapter struct{ svc *userSvc.UserService }

func (u *userSvcAdapter) GetByID(ctx context.Context, id string) (*userentity.User, error) {
	return u.svc.GetByID(ctx, id)
}

// asynqPublisherAdapter adapts *worker.Client — whose Enqueue takes a
// variadic ...asynq.Option the transport-neutral contract doesn't have — to
// queue.Publisher, so the Redis backend can participate in pkg/queue/router
// alongside RabbitMQ/Kafka (which already satisfy queue.Publisher natively).
type asynqPublisherAdapter struct{ c *worker.Client }

func (a asynqPublisherAdapter) Enqueue(taskType string, payload any) error {
	return a.c.Enqueue(taskType, payload)
}

func (a asynqPublisherAdapter) EnqueueEmail(to, subject, html, text string) error {
	return a.c.EnqueueEmail(to, subject, html, text)
}

func (a asynqPublisherAdapter) EnqueuePromotionalEmail(to, subject, html, text string) error {
	return a.c.EnqueuePromotionalEmail(to, subject, html, text)
}

func (a asynqPublisherAdapter) Close() error { return a.c.Close() }

var _ queue.Publisher = asynqPublisherAdapter{}
