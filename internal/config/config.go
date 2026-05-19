// Package config loads runtime configuration from environment variables.
// All keys are prefixed with ONEFACTURE_ per the contribution guide.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env      string
	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Sidecar  SidecarConfig
	Auth     AuthConfig
}

type HTTPConfig struct {
	Addr            string
	PublicBaseURL   string
	TrustedProxies  []string
	RateLimitPerMin int
}

type DatabaseConfig struct {
	DSN            string
	MaxConns       int32
	MigrationsPath string
	ConnectTimeout time.Duration
	StatementCache bool
}

type RedisConfig struct {
	Addr      string
	Password  string
	DB        int
	StreamKey string
}

type SidecarConfig struct {
	BaseURL string
	Timeout time.Duration
}

type AuthConfig struct {
	BootstrapAPIKey string // for first-time setup; if empty, no bootstrap key
	HashPepper      string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env: env("ONEFACTURE_ENV", "development"),
		HTTP: HTTPConfig{
			Addr:            env("ONEFACTURE_HTTP_ADDR", ":8080"),
			PublicBaseURL:   env("ONEFACTURE_PUBLIC_BASE_URL", "http://localhost:8080"),
			RateLimitPerMin: envInt("ONEFACTURE_RATE_LIMIT_PER_MIN", 600),
		},
		Database: DatabaseConfig{
			DSN:            env("ONEFACTURE_DB_DSN", "postgres://onefacture:onefacture@localhost:5432/onefacture?sslmode=disable"),
			MaxConns:       envInt32("ONEFACTURE_DB_MAX_CONNS", 20),
			MigrationsPath: env("ONEFACTURE_DB_MIGRATIONS_PATH", "internal/storage/migrations"),
			ConnectTimeout: envDuration("ONEFACTURE_DB_CONNECT_TIMEOUT", 5*time.Second),
			StatementCache: envBool("ONEFACTURE_DB_STATEMENT_CACHE", true),
		},
		Redis: RedisConfig{
			Addr:      env("ONEFACTURE_REDIS_ADDR", "localhost:6379"),
			Password:  env("ONEFACTURE_REDIS_PASSWORD", ""),
			DB:        envInt("ONEFACTURE_REDIS_DB", 0),
			StreamKey: env("ONEFACTURE_REDIS_STREAM", "onefacture.events"),
		},
		Sidecar: SidecarConfig{
			BaseURL: env("ONEFACTURE_SIDECAR_URL", "http://localhost:8081"),
			Timeout: envDuration("ONEFACTURE_SIDECAR_TIMEOUT", 30*time.Second),
		},
		Auth: AuthConfig{
			BootstrapAPIKey: env("ONEFACTURE_BOOTSTRAP_API_KEY", ""),
			HashPepper:      env("ONEFACTURE_HASH_PEPPER", ""),
		},
	}
	if cfg.Auth.HashPepper == "" && cfg.Env != "development" {
		return nil, fmt.Errorf("ONEFACTURE_HASH_PEPPER must be set in non-development environments")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envInt32(key string, fallback int32) int32 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			return int32(n)
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
