package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Momo     MomoConfig
	Airtel   AirtelConfig
	R2       R2Config
	Email    EmailConfig
}

type AppConfig struct {
	Name string
	Env  string
	Port int
}

type DatabaseConfig struct {
	Driver          string
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type EmailConfig struct {
	ResendAPIKey string
	From         string
	ReplyTo      string
	WebBaseURL   string
}

func (e EmailConfig) Enabled() bool {
	return e.ResendAPIKey != "" && e.From != ""
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type MomoConfig struct {
	BaseURL         string
	SubscriptionKey string
	APIUser         string
	APIKey          string
	TargetEnv       string
	CallbackURL     string
	Currency        string
}

func (m MomoConfig) Enabled() bool {
	return m.BaseURL != "" && m.SubscriptionKey != "" && m.APIUser != "" && m.APIKey != ""
}

type AirtelConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	CallbackURL  string
}

func (a AirtelConfig) Enabled() bool {
	return a.BaseURL != "" && a.ClientID != "" && a.ClientSecret != ""
}

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string
	Endpoint        string
}

func (r R2Config) Enabled() bool {
	return r.AccessKeyID != "" && r.SecretAccessKey != "" && r.Bucket != "" && r.Endpoint != ""
}

func Load() (*Config, error) {
	if err := loadDotEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "[config] warning: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "[config] .env loaded")
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath(".")

	v.SetDefault("app.name", "rogele")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", 8080)
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.host", "127.0.0.1")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("jwt.access_ttl", "15m")
	v.SetDefault("jwt.refresh_ttl", "168h")
	v.SetDefault("momo.currency", "EUR")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	applyStringEnv(&cfg)
	applyIntEnv(&cfg)

	accessTTL := getStringEnv("ROGELEDB_JWT_ACCESS_TTL", v.GetString("jwt.access_ttl"))
	refreshTTL := getStringEnv("ROGELEDB_JWT_REFRESH_TTL", v.GetString("jwt.refresh_ttl"))
	connLife := getStringEnv("ROGELEDB_DB_CONN_MAX_LIFETIME", v.GetString("database.conn_max_lifetime"))

	access, err := time.ParseDuration(accessTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid jwt.access_ttl: %w", err)
	}
	refresh, err := time.ParseDuration(refreshTTL)
	if err != nil {
		return nil, fmt.Errorf("invalid jwt.refresh_ttl: %w", err)
	}
	life, err := time.ParseDuration(connLife)
	if err != nil {
		return nil, fmt.Errorf("invalid database.conn_max_lifetime: %w", err)
	}
	cfg.JWT.AccessTTL = access
	cfg.JWT.RefreshTTL = refresh
	cfg.Database.ConnMaxLifetime = life

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT secret is empty — set ROGELEDB_JWT_SECRET in .env")
	}

	if cfg.Momo.Currency == "" {
		cfg.Momo.Currency = "EUR"
	}

	if cfg.App.Env == "development" {
		fmt.Fprintf(os.Stderr,
			"[config] env=%s db=%s:%d/%s user=%s max_open=%d max_idle=%d momo=%v airtel=%v r2=%v email=%v\n",
			cfg.App.Env,
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
			cfg.Database.User,
			cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns,
			cfg.Momo.Enabled(),
			cfg.Airtel.Enabled(),
			cfg.R2.Enabled(),
			cfg.Email.Enabled(),
		)
	}

	return &cfg, nil
}

func applyStringEnv(cfg *Config) {
	set := func(key string, dst *string) {
		if v := getStringEnv(key, ""); v != "" {
			*dst = v
		}
	}
	set("ROGELEDB_APP_NAME", &cfg.App.Name)
	set("ROGELEDB_APP_ENV", &cfg.App.Env)
	set("ROGELEDB_DB_DRIVER", &cfg.Database.Driver)
	set("ROGELEDB_DB_HOST", &cfg.Database.Host)
	set("ROGELEDB_DB_USER", &cfg.Database.User)
	set("ROGELEDB_DB_PASSWORD", &cfg.Database.Password)
	set("ROGELEDB_DB_NAME", &cfg.Database.Name)
	set("ROGELEDB_DB_SSLMODE", &cfg.Database.SSLMode)
	set("ROGELEDB_JWT_SECRET", &cfg.JWT.Secret)
	set("ROGELEDB_MTN_MOMO_BASE_URL", &cfg.Momo.BaseURL)
	set("ROGELEDB_MTN_MOMO_SUBSCRIPTION_KEY", &cfg.Momo.SubscriptionKey)
	set("ROGELEDB_MTN_MOMO_API_USER", &cfg.Momo.APIUser)
	set("ROGELEDB_MTN_MOMO_API_KEY", &cfg.Momo.APIKey)
	set("ROGELEDB_MTN_MOMO_TARGET_ENV", &cfg.Momo.TargetEnv)
	set("ROGELEDB_MTN_MOMO_CALLBACK_URL", &cfg.Momo.CallbackURL)
	set("ROGELEDB_MTN_MOMO_CURRENCY", &cfg.Momo.Currency)
	set("ROGELEDB_AIRTEL_BASE_URL", &cfg.Airtel.BaseURL)
	set("ROGELEDB_AIRTEL_CLIENT_ID", &cfg.Airtel.ClientID)
	set("ROGELEDB_AIRTEL_CLIENT_SECRET", &cfg.Airtel.ClientSecret)
	set("ROGELEDB_AIRTEL_CALLBACK_URL", &cfg.Airtel.CallbackURL)
	set("ROGELEDB_R2_ACCOUNT_ID", &cfg.R2.AccountID)
	set("ROGELEDB_R2_ACCESS_KEY_ID", &cfg.R2.AccessKeyID)
	set("ROGELEDB_R2_SECRET_ACCESS_KEY", &cfg.R2.SecretAccessKey)
	set("ROGELEDB_R2_BUCKET", &cfg.R2.Bucket)
	set("ROGELEDB_R2_PUBLIC_URL", &cfg.R2.PublicURL)
	set("ROGELEDB_R2_ENDPOINT", &cfg.R2.Endpoint)
	set("ROGELEDB_RESEND_API_KEY", &cfg.Email.ResendAPIKey)
	set("ROGELEDB_EMAIL_FROM", &cfg.Email.From)
	set("ROGELEDB_EMAIL_REPLY_TO", &cfg.Email.ReplyTo)
	set("ROGELEDB_WEB_BASE_URL", &cfg.Email.WebBaseURL)
}

func applyIntEnv(cfg *Config) {
	if v := getIntEnv("ROGELEDB_APP_PORT"); v > 0 {
		cfg.App.Port = v
	}
	if v := getIntEnv("ROGELEDB_DB_PORT"); v > 0 {
		cfg.Database.Port = v
	}
	if v := getIntEnv("ROGELEDB_DB_MAX_OPEN_CONNS"); v > 0 {
		cfg.Database.MaxOpenConns = v
	}
	if v := getIntEnv("ROGELEDB_DB_MAX_IDLE_CONNS"); v > 0 {
		cfg.Database.MaxIdleConns = v
	}
}

func getStringEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return fallback
}

func getIntEnv(key string) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return 0
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func loadDotEnv() error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			if err := godotenv.Load(candidate); err != nil {
				return fmt.Errorf("load %s: %w", candidate, err)
			}
			return nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("no .env file found in %s or any parent", dir)
		}
		dir = parent
	}
}
