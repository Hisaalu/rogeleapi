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

	if cfg.App.Env == "development" {
		fmt.Fprintf(os.Stderr,
			"[config] env=%s db=%s:%d/%s user=%s max_open=%d max_idle=%d\n",
			cfg.App.Env,
			cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
			cfg.Database.User,
			cfg.Database.MaxOpenConns, cfg.Database.MaxIdleConns,
		)
	}

	return &cfg, nil
}

func applyStringEnv(cfg *Config) {
	if v := getStringEnv("ROGELEDB_APP_NAME", ""); v != "" {
		cfg.App.Name = v
	}
	if v := getStringEnv("ROGELEDB_APP_ENV", ""); v != "" {
		cfg.App.Env = v
	}
	if v := getStringEnv("ROGELEDB_DB_DRIVER", ""); v != "" {
		cfg.Database.Driver = v
	}
	if v := getStringEnv("ROGELEDB_DB_HOST", ""); v != "" {
		cfg.Database.Host = v
	}
	if v := getStringEnv("ROGELEDB_DB_USER", ""); v != "" {
		cfg.Database.User = v
	}
	if v := getStringEnv("ROGELEDB_DB_PASSWORD", ""); v != "" {
		cfg.Database.Password = v
	}
	if v := getStringEnv("ROGELEDB_DB_NAME", ""); v != "" {
		cfg.Database.Name = v
	}
	if v := getStringEnv("ROGELEDB_DB_SSLMODE", ""); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := getStringEnv("ROGELEDB_JWT_SECRET", ""); v != "" {
		cfg.JWT.Secret = v
	}
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
