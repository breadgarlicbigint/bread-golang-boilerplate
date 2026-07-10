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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger   "github.com/swaggo/gin-swagger"
	jwtpkg       "github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/jwt"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/sms"
)

// App holds all wired-up dependencies and the HTTP server.
type App struct {
	cfg    *config.Config
	log    *zap.Logger
	mongo  *database.MongoDB
	rdb    *redis.Client
	engine *gin.Engine
	server *http.Server
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

	// ── Core modules ──────────────────────────────────────────────────────────
	uRepo       := userRepo.New(mongo)
	rRepo       := roleRepo.New(mongo)
	rSvc        := roleSvc.New(rRepo)
	uSvc        := userSvc.New(uRepo, hasher, cfg.Auth, localMailer, cfg.App.URL, log)
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
	notifService := notifSvc.New(mongo, fcmSender, mailer, log)

	// ── App versioning ────────────────────────────────────────────────────────
	versionService := appverSvc.New(mongo)

	// ── Analytics ─────────────────────────────────────────────────────────────
	analyticsService := analyticsSvc.New(mongo)

	// ── Middleware ────────────────────────────────────────────────────────────
	sessionStore := &redisSessionStore{rdb}
	authMw       := middleware.AuthJWTAccess(jwtMgr, sessionStore)
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

	// App versioning
	appverHdl.New(versionService).RegisterRoutes(v1, authMw, adminMw)

	// Analytics (admin-only)
	adminV1 := engine.Group("/v1", tenantResolver, authMw, adminMw)
	analyticsHdl.New(analyticsService, rdb, cfg.Auth.MaxPasswordAttempts).RegisterRoutes(adminV1)

	app := &App{
		cfg:    cfg,
		log:    log,
		mongo:  mongo,
		rdb:    rdb,
		engine: engine,
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
