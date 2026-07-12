package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App          AppConfig
	Mongo        MongoConfig
	Redis        RedisConfig
	JWT          JWTConfig
	Auth         AuthConfig
	Rate         RateLimitConfig
	APIKey       APIKeyConfig
	AWS          AWSConfig
	Mail         MailConfig
	Google       GoogleConfig
	Apple        AppleConfig
	GitHub       GitHubConfig
	WebAuthn     WebAuthnConfig
	Twilio       TwilioConfig
	Firebase     FirebaseConfig
	I18n         I18nConfig
	Tenant       TenantConfig
	Notification NotificationConfig
	Sentry       SentryConfig
	Worker       WorkerConfig
	Queue        QueueConfig
	RabbitMQ     RabbitMQConfig
	Kafka        KafkaConfig
}

type AppConfig struct {
	Name     string        `mapstructure:"APP_NAME"`
	Env      string        `mapstructure:"APP_ENV"`
	Port     string        `mapstructure:"APP_PORT"`
	Version  string        `mapstructure:"APP_VERSION"`
	Debug    bool          `mapstructure:"APP_DEBUG"`
	Timezone string        `mapstructure:"APP_TIMEZONE"`
	Timeout  time.Duration `mapstructure:"APP_TIMEOUT"`
	URL      string        `mapstructure:"APP_URL"` // frontend base URL used to build links in outbound emails
}

type MongoConfig struct {
	URI      string        `mapstructure:"MONGO_URI"`
	DBName   string        `mapstructure:"MONGO_DB_NAME"`
	IDType   string        `mapstructure:"MONGO_ID_TYPE"` // "uuid" (default) or "objectid"
	PoolMin  uint64        `mapstructure:"MONGO_POOL_MIN"`
	PoolMax  uint64        `mapstructure:"MONGO_POOL_MAX"`
	Timeout  time.Duration `mapstructure:"MONGO_TIMEOUT"`
}

type RedisConfig struct {
	Host     string `mapstructure:"REDIS_HOST"`
	Port     string `mapstructure:"REDIS_PORT"`
	Password string `mapstructure:"REDIS_PASSWORD"`
	DB       int    `mapstructure:"REDIS_DB"`
	TLS      bool   `mapstructure:"REDIS_TLS"`
}

type JWTConfig struct {
	AccessPrivateKeyPath  string        `mapstructure:"JWT_ACCESS_PRIVATE_KEY_PATH"`
	AccessPublicKeyPath   string        `mapstructure:"JWT_ACCESS_PUBLIC_KEY_PATH"`
	AccessExpire          time.Duration `mapstructure:"JWT_ACCESS_EXPIRE"`
	RefreshPrivateKeyPath string        `mapstructure:"JWT_REFRESH_PRIVATE_KEY_PATH"`
	RefreshPublicKeyPath  string        `mapstructure:"JWT_REFRESH_PUBLIC_KEY_PATH"`
	RefreshExpire         time.Duration `mapstructure:"JWT_REFRESH_EXPIRE"`
}

type AuthConfig struct {
	PasswordMinLength    int           `mapstructure:"AUTH_PASSWORD_MIN_LENGTH"`
	MaxPasswordAttempts  int           `mapstructure:"AUTH_MAX_PASSWORD_ATTEMPTS"`
	LockoutDuration      time.Duration `mapstructure:"AUTH_LOCKOUT_DURATION"`
}

type RateLimitConfig struct {
	Requests int           `mapstructure:"RATE_LIMIT_REQUESTS"`
	Period   time.Duration `mapstructure:"RATE_LIMIT_PERIOD"`
}

type APIKeyConfig struct {
	Header string `mapstructure:"API_KEY_HEADER"`
	Prefix string `mapstructure:"API_KEY_PREFIX"`
}

type AWSConfig struct {
	Region          string `mapstructure:"AWS_REGION"`
	AccessKeyID     string `mapstructure:"AWS_ACCESS_KEY_ID"`
	SecretAccessKey string `mapstructure:"AWS_SECRET_ACCESS_KEY"`
	S3              AWSS3Config
	SES             AWSSESConfig
}

type AWSS3Config struct {
	Bucket        string        `mapstructure:"AWS_S3_BUCKET"`
	PresignExpire time.Duration `mapstructure:"AWS_S3_PRESIGN_EXPIRE"`
	BaseURL       string        `mapstructure:"AWS_S3_BASE_URL"`
}

type AWSSESConfig struct {
	FromEmail string `mapstructure:"AWS_SES_FROM_EMAIL"`
	FromName  string `mapstructure:"AWS_SES_FROM_NAME"`
}

// MailConfig selects the email transport. Driver: "ses" (default, uses
// AWSConfig/AWSSESConfig above) or "smtp" (uses SMTP below).
type MailConfig struct {
	Driver string     `mapstructure:"MAIL_DRIVER"`
	SMTP   SMTPConfig
}

type SMTPConfig struct {
	Host      string `mapstructure:"SMTP_HOST"`
	Port      string `mapstructure:"SMTP_PORT"`
	Username  string `mapstructure:"SMTP_USERNAME"`
	Password  string `mapstructure:"SMTP_PASSWORD"`
	FromEmail string `mapstructure:"SMTP_FROM_EMAIL"`
	FromName  string `mapstructure:"SMTP_FROM_NAME"`
}

type GoogleConfig struct {
	ClientID     string `mapstructure:"GOOGLE_CLIENT_ID"`
	ClientSecret string `mapstructure:"GOOGLE_CLIENT_SECRET"`
	RedirectURL  string `mapstructure:"GOOGLE_REDIRECT_URL"`
}

type AppleConfig struct {
	ClientID       string `mapstructure:"APPLE_CLIENT_ID"`
	TeamID         string `mapstructure:"APPLE_TEAM_ID"`
	KeyID          string `mapstructure:"APPLE_KEY_ID"`
	PrivateKeyPath string `mapstructure:"APPLE_PRIVATE_KEY_PATH"`
}

type SentryConfig struct {
	DSN               string  `mapstructure:"SENTRY_DSN"`
	TracesSampleRate  float64 `mapstructure:"SENTRY_TRACES_SAMPLE_RATE"`
}

type WorkerConfig struct {
	Concurrency int    `mapstructure:"WORKER_CONCURRENCY"`
	Queues      string `mapstructure:"WORKER_QUEUES"`
}

// QueueConfig selects which transport the API enqueues background jobs onto.
// It MUST match the worker that is actually running:
//
//	redis    → apps/worker          (Redis/Asynq)   — default
//	rabbitmq → apps/worker-rabbitmq  (pkg/queue/rabbitmq)
//	kafka    → apps/worker-kafka     (pkg/queue/kafka)
//
// A mismatch (e.g. QUEUE_DRIVER=redis while only the RabbitMQ worker runs)
// means jobs are published to a broker nobody consumes, so they never run.
type QueueConfig struct {
	Driver string `mapstructure:"QUEUE_DRIVER"` // redis | rabbitmq | kafka
}

// RabbitMQConfig configures the AMQP queue-job/worker backend (pkg/queue/rabbitmq).
type RabbitMQConfig struct {
	URL      string `mapstructure:"RABBITMQ_URL"`      // amqp://user:pass@host:5672/
	Exchange string `mapstructure:"RABBITMQ_EXCHANGE"` // durable direct exchange for jobs
	Queue    string `mapstructure:"RABBITMQ_QUEUE"`    // durable work queue the worker consumes
	Prefetch int    `mapstructure:"RABBITMQ_PREFETCH"` // unacked messages in flight / worker concurrency
}

// KafkaConfig configures the Kafka queue-job/worker backend (pkg/queue/kafka).
type KafkaConfig struct {
	Brokers string `mapstructure:"KAFKA_BROKERS"`  // comma-separated host:port list
	Topic   string `mapstructure:"KAFKA_TOPIC"`    // topic jobs are produced to / consumed from
	GroupID string `mapstructure:"KAFKA_GROUP_ID"` // consumer group id for the worker
}

func Load() (*Config, error) {
	v := viper.New()

	// BREAD_CONFIG_FILE lets make targets (seed-local, migrate-indexes-local)
	// or CI pipelines point at a custom .env path:
	//   BREAD_CONFIG_FILE=/path/to/.env.staging make seed-local
	if cf := os.Getenv("BREAD_CONFIG_FILE"); cf != "" {
		v.SetConfigFile(cf)
	} else {
		v.SetConfigName(".env")
		v.SetConfigType("env")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Defaults
	v.SetDefault("APP_NAME", "bread-golang-boilerplate")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", "3000")
	v.SetDefault("APP_VERSION", "1.0.0")
	v.SetDefault("APP_DEBUG", true)
	v.SetDefault("APP_TIMEZONE", "UTC")
	v.SetDefault("APP_TIMEOUT", "30s")
	v.SetDefault("APP_URL", "http://localhost:5173") // web/ test client — used to build links in outbound emails
	v.SetDefault("MONGO_URI", "mongodb://localhost:27017")  // Docker overrides this via environment
	v.SetDefault("MONGO_ID_TYPE", "uuid")                 // "uuid" or "objectid" — see docs/id-migration.md
	v.SetDefault("MONGO_DB_NAME", "bread_boilerplate")
	v.SetDefault("MONGO_POOL_MIN", 2)
	v.SetDefault("MONGO_POOL_MAX", 10)
	v.SetDefault("MONGO_TIMEOUT", "30s") // 30s for replica set election during startup
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("JWT_ACCESS_EXPIRE", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRE", "168h") // 7 days
	v.SetDefault("AUTH_PASSWORD_MIN_LENGTH", 8)
	v.SetDefault("AUTH_MAX_PASSWORD_ATTEMPTS", 5)
	v.SetDefault("AUTH_LOCKOUT_DURATION", "15m")
	v.SetDefault("RATE_LIMIT_REQUESTS", 100)
	v.SetDefault("RATE_LIMIT_PERIOD", "1m")
	v.SetDefault("API_KEY_HEADER", "X-Api-Key")
	v.SetDefault("API_KEY_PREFIX", "bread")
	v.SetDefault("MAIL_DRIVER", "ses") // "ses" or "smtp"
	v.SetDefault("SMTP_PORT", "587")
	v.SetDefault("WORKER_CONCURRENCY", 10)
	v.SetDefault("WORKER_QUEUES", "default:6,critical:3,low:1")
	v.SetDefault("QUEUE_DRIVER", "redis") // redis | rabbitmq | kafka
	// RabbitMQ (pkg/queue/rabbitmq)
	v.SetDefault("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	v.SetDefault("RABBITMQ_EXCHANGE", "bread.tasks")
	v.SetDefault("RABBITMQ_QUEUE", "bread.worker")
	v.SetDefault("RABBITMQ_PREFETCH", 10)
	// Kafka (pkg/queue/kafka)
	v.SetDefault("KAFKA_BROKERS", "localhost:9092")
	v.SetDefault("KAFKA_TOPIC", "bread.tasks")
	v.SetDefault("KAFKA_GROUP_ID", "bread.worker")
	v.SetDefault("GITHUB_REDIRECT_URL", "http://localhost:3000/v1/auth/github/callback")
	v.SetDefault("WEBAUTHN_RP_ID", "localhost")
	v.SetDefault("WEBAUTHN_RP_ORIGIN", "http://localhost:3000")
	v.SetDefault("WEBAUTHN_RP_NAME", "Bread Boilerplate")
	v.SetDefault("DEFAULT_LANG", "en")
	v.SetDefault("LOCALES_DIR", "./locales")
	v.SetDefault("MULTI_TENANT_ENABLED", false)
	v.SetDefault("BASE_DOMAIN", "localhost")
	v.SetDefault("NOTIFICATION_ENABLED", true)
	v.SetDefault("APP_VERSION_CHECK_ENABLED", true)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: read error: %w", err)
		}
	}

	cfg := &Config{}

	cfg.App = AppConfig{
		Name:    v.GetString("APP_NAME"),
		Env:     v.GetString("APP_ENV"),
		Port:    v.GetString("APP_PORT"),
		Version: v.GetString("APP_VERSION"),
		Debug:   v.GetBool("APP_DEBUG"),
		Timezone: v.GetString("APP_TIMEZONE"),
		URL:     v.GetString("APP_URL"),
	}
	cfg.Mongo = MongoConfig{
		URI:     v.GetString("MONGO_URI"),
		DBName:  v.GetString("MONGO_DB_NAME"),
		IDType:  v.GetString("MONGO_ID_TYPE"),
		PoolMin: v.GetUint64("MONGO_POOL_MIN"),
		PoolMax: v.GetUint64("MONGO_POOL_MAX"),
		Timeout: v.GetDuration("MONGO_TIMEOUT"),
	}
	cfg.Redis = RedisConfig{
		Host:     v.GetString("REDIS_HOST"),
		Port:     v.GetString("REDIS_PORT"),
		Password: v.GetString("REDIS_PASSWORD"),
		DB:       v.GetInt("REDIS_DB"),
		TLS:      v.GetBool("REDIS_TLS"),
	}
	cfg.JWT = JWTConfig{
		AccessPrivateKeyPath:  v.GetString("JWT_ACCESS_PRIVATE_KEY_PATH"),
		AccessPublicKeyPath:   v.GetString("JWT_ACCESS_PUBLIC_KEY_PATH"),
		AccessExpire:          v.GetDuration("JWT_ACCESS_EXPIRE"),
		RefreshPrivateKeyPath: v.GetString("JWT_REFRESH_PRIVATE_KEY_PATH"),
		RefreshPublicKeyPath:  v.GetString("JWT_REFRESH_PUBLIC_KEY_PATH"),
		RefreshExpire:         v.GetDuration("JWT_REFRESH_EXPIRE"),
	}
	cfg.Auth = AuthConfig{
		PasswordMinLength:   v.GetInt("AUTH_PASSWORD_MIN_LENGTH"),
		MaxPasswordAttempts: v.GetInt("AUTH_MAX_PASSWORD_ATTEMPTS"),
		LockoutDuration:     v.GetDuration("AUTH_LOCKOUT_DURATION"),
	}
	cfg.Rate = RateLimitConfig{
		Requests: v.GetInt("RATE_LIMIT_REQUESTS"),
		Period:   v.GetDuration("RATE_LIMIT_PERIOD"),
	}
	cfg.APIKey = APIKeyConfig{
		Header: v.GetString("API_KEY_HEADER"),
		Prefix: v.GetString("API_KEY_PREFIX"),
	}
	cfg.AWS = AWSConfig{
		Region:          v.GetString("AWS_REGION"),
		AccessKeyID:     v.GetString("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: v.GetString("AWS_SECRET_ACCESS_KEY"),
		S3: AWSS3Config{
			Bucket:        v.GetString("AWS_S3_BUCKET"),
			PresignExpire: v.GetDuration("AWS_S3_PRESIGN_EXPIRE"),
			BaseURL:       v.GetString("AWS_S3_BASE_URL"),
		},
		SES: AWSSESConfig{
			FromEmail: v.GetString("AWS_SES_FROM_EMAIL"),
			FromName:  v.GetString("AWS_SES_FROM_NAME"),
		},
	}
	cfg.Mail = MailConfig{
		Driver: v.GetString("MAIL_DRIVER"),
		SMTP: SMTPConfig{
			Host:      v.GetString("SMTP_HOST"),
			Port:      v.GetString("SMTP_PORT"),
			Username:  v.GetString("SMTP_USERNAME"),
			Password:  v.GetString("SMTP_PASSWORD"),
			FromEmail: v.GetString("SMTP_FROM_EMAIL"),
			FromName:  v.GetString("SMTP_FROM_NAME"),
		},
	}
	cfg.Google = GoogleConfig{
		ClientID:     v.GetString("GOOGLE_CLIENT_ID"),
		ClientSecret: v.GetString("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  v.GetString("GOOGLE_REDIRECT_URL"),
	}
	cfg.Apple = AppleConfig{
		ClientID:       v.GetString("APPLE_CLIENT_ID"),
		TeamID:         v.GetString("APPLE_TEAM_ID"),
		KeyID:          v.GetString("APPLE_KEY_ID"),
		PrivateKeyPath: v.GetString("APPLE_PRIVATE_KEY_PATH"),
	}
	cfg.Sentry = SentryConfig{
		DSN:              v.GetString("SENTRY_DSN"),
		TracesSampleRate: v.GetFloat64("SENTRY_TRACES_SAMPLE_RATE"),
	}
	cfg.Worker = WorkerConfig{
		Concurrency: v.GetInt("WORKER_CONCURRENCY"),
		Queues:      v.GetString("WORKER_QUEUES"),
	}
	cfg.Queue = QueueConfig{
		Driver: v.GetString("QUEUE_DRIVER"),
	}
	cfg.RabbitMQ = RabbitMQConfig{
		URL:      v.GetString("RABBITMQ_URL"),
		Exchange: v.GetString("RABBITMQ_EXCHANGE"),
		Queue:    v.GetString("RABBITMQ_QUEUE"),
		Prefetch: v.GetInt("RABBITMQ_PREFETCH"),
	}
	cfg.Kafka = KafkaConfig{
		Brokers: v.GetString("KAFKA_BROKERS"),
		Topic:   v.GetString("KAFKA_TOPIC"),
		GroupID: v.GetString("KAFKA_GROUP_ID"),
	}
	cfg.GitHub = GitHubConfig{
		ClientID:     v.GetString("GITHUB_CLIENT_ID"),
		ClientSecret: v.GetString("GITHUB_CLIENT_SECRET"),
		RedirectURL:  v.GetString("GITHUB_REDIRECT_URL"),
	}
	cfg.WebAuthn = WebAuthnConfig{
		RPID:     v.GetString("WEBAUTHN_RP_ID"),
		RPOrigin: v.GetString("WEBAUTHN_RP_ORIGIN"),
		RPName:   v.GetString("WEBAUTHN_RP_NAME"),
	}
	cfg.Twilio = TwilioConfig{
		AccountSID:   v.GetString("TWILIO_ACCOUNT_SID"),
		AuthToken:    v.GetString("TWILIO_AUTH_TOKEN"),
		FromSMS:      v.GetString("TWILIO_FROM_SMS"),
		FromWhatsApp: v.GetString("TWILIO_FROM_WHATSAPP"),
	}
	cfg.Firebase = FirebaseConfig{
		CredentialsFile: v.GetString("FIREBASE_CREDENTIALS_FILE"),
	}
	cfg.I18n = I18nConfig{
		DefaultLang: v.GetString("DEFAULT_LANG"),
		LocalesDir:  v.GetString("LOCALES_DIR"),
	}
	cfg.Tenant = TenantConfig{
		Enabled:    v.GetBool("MULTI_TENANT_ENABLED"),
		BaseDomain: v.GetString("BASE_DOMAIN"),
	}
	cfg.Notification = NotificationConfig{
		Enabled: v.GetBool("NOTIFICATION_ENABLED"),
	}

	return cfg, nil
}


// ── Appended for new features ─────────────────────────────────────────────────

type GitHubConfig struct {
	ClientID     string `mapstructure:"GITHUB_CLIENT_ID"`
	ClientSecret string `mapstructure:"GITHUB_CLIENT_SECRET"`
	RedirectURL  string `mapstructure:"GITHUB_REDIRECT_URL"`
}

type WebAuthnConfig struct {
	RPID     string `mapstructure:"WEBAUTHN_RP_ID"`
	RPOrigin string `mapstructure:"WEBAUTHN_RP_ORIGIN"`
	RPName   string `mapstructure:"WEBAUTHN_RP_NAME"`
}

type TwilioConfig struct {
	AccountSID   string `mapstructure:"TWILIO_ACCOUNT_SID"`
	AuthToken    string `mapstructure:"TWILIO_AUTH_TOKEN"`
	FromSMS      string `mapstructure:"TWILIO_FROM_SMS"`
	FromWhatsApp string `mapstructure:"TWILIO_FROM_WHATSAPP"`
}

type TenantConfig struct {
	Enabled    bool   `mapstructure:"MULTI_TENANT_ENABLED"`
	BaseDomain string `mapstructure:"BASE_DOMAIN"`
}

type VersioningConfig struct {
	Enabled bool `mapstructure:"APP_VERSION_CHECK_ENABLED"`
}

type FirebaseConfig struct {
	CredentialsFile string `mapstructure:"FIREBASE_CREDENTIALS_FILE"`
}

type I18nConfig struct {
	DefaultLang string `mapstructure:"DEFAULT_LANG"`
	LocalesDir  string `mapstructure:"LOCALES_DIR"`
}

type NotificationConfig struct {
	Enabled bool `mapstructure:"NOTIFICATION_ENABLED"`
}
