package config

import (
	"fmt"
	"os"
	"path/filepath"
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

	v.SetEnvPrefix("ROGELEDB")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	explicit := map[string]string{
		"app.name":                   "ROGELEDB_APP_NAME",
		"app.env":                    "ROGELEDB_APP_ENV",
		"app.port":                   "ROGELEDB_APP_PORT",
		"database.driver":            "ROGELEDB_DB_DRIVER",
		"database.host":              "ROGELEDB_DB_HOST",
		"database.port":              "ROGELEDB_DB_PORT",
		"database.user":              "ROGELEDB_DB_USER",
		"database.password":          "ROGELEDB_DB_PASSWORD",
		"database.name":              "ROGELEDB_DB_NAME",
		"database.sslmode":           "ROGELEDB_DB_SSLMODE",
		"database.max_open_conns":    "ROGELEDB_DB_MAX_OPEN_CONNS",
		"database.max_idle_conns":    "ROGELEDB_DB_MAX_IDLE_CONNS",
		"database.conn_max_lifetime": "ROGELEDB_DB_CONN_MAX_LIFETIME",
		"jwt.secret":                 "ROGELEDB_JWT_SECRET",
		"jwt.access_ttl":             "ROGELEDB_JWT_ACCESS_TTL",
		"jwt.refresh_ttl":            "ROGELEDB_JWT_REFRESH_TTL",
	}
	for key, env := range explicit {
		_ = v.BindEnv(key, env)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	access, err := time.ParseDuration(v.GetString("jwt.access_ttl"))
	if err != nil {
		return nil, fmt.Errorf("invalid jwt.access_ttl: %w", err)
	}
	refresh, err := time.ParseDuration(v.GetString("jwt.refresh_ttl"))
	if err != nil {
		return nil, fmt.Errorf("invalid jwt.refresh_ttl: %w", err)
	}
	cfg.JWT.AccessTTL = access
	cfg.JWT.RefreshTTL = refresh

	connLife, err := time.ParseDuration(v.GetString("database.conn_max_lifetime"))
	if err != nil {
		return nil, fmt.Errorf("invalid database.conn_max_lifetime: %w", err)
	}
	cfg.Database.ConnMaxLifetime = connLife

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT secret is empty — set ROGELEDB_JWT_SECRET in .env")
	}

	if cfg.App.Env == "development" {
		fmt.Fprintf(os.Stderr,
			"[config] env=%s db=%s:%d/%s user=%s\n",
			cfg.App.Env, cfg.Database.Host, cfg.Database.Port,
			cfg.Database.Name, cfg.Database.User,
		)
	}

	return &cfg, nil
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
