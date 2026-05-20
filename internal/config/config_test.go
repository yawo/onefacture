package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars
	clearEnv()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Check defaults
	require.Equal(t, "development", cfg.Env)
	require.Equal(t, ":8080", cfg.HTTP.Addr)
	require.Equal(t, "http://localhost:8080", cfg.HTTP.PublicBaseURL)
	require.Equal(t, 600, cfg.HTTP.RateLimitPerMin)
	require.Equal(t, "postgres://onefacture:onefacture@localhost:5432/onefacture?sslmode=disable", cfg.Database.DSN)
	require.Equal(t, int32(20), cfg.Database.MaxConns)
	require.Equal(t, "internal/storage/migrations", cfg.Database.MigrationsPath)
	require.Equal(t, 5*time.Second, cfg.Database.ConnectTimeout)
	require.True(t, cfg.Database.StatementCache)
	require.Equal(t, "localhost:6379", cfg.Redis.Addr)
	require.Equal(t, "", cfg.Redis.Password)
	require.Equal(t, 0, cfg.Redis.DB)
	require.Equal(t, "onefacture.events", cfg.Redis.StreamKey)
	require.Equal(t, "http://localhost:8081", cfg.Sidecar.BaseURL)
	require.Equal(t, 30*time.Second, cfg.Sidecar.Timeout)
	require.Equal(t, "", cfg.Auth.BootstrapAPIKey)
	require.Equal(t, "", cfg.Auth.HashPepper)
}

func TestLoadFromEnv(t *testing.T) {
	// Clear any env vars
	clearEnv()

	// Set custom env vars
	os.Setenv("ONEFACTURE_ENV", "production")
	os.Setenv("ONEFACTURE_HTTP_ADDR", ":9000")
	os.Setenv("ONEFACTURE_PUBLIC_BASE_URL", "https://api.example.com")
	os.Setenv("ONEFACTURE_RATE_LIMIT_PER_MIN", "1000")
	os.Setenv("ONEFACTURE_DB_DSN", "postgres://prod:secret@db.example.com/prod")
	os.Setenv("ONEFACTURE_DB_MAX_CONNS", "50")
	os.Setenv("ONEFACTURE_DB_MIGRATIONS_PATH", "migrations/prod")
	os.Setenv("ONEFACTURE_DB_CONNECT_TIMEOUT", "10s")
	os.Setenv("ONEFACTURE_DB_STATEMENT_CACHE", "false")
	os.Setenv("ONEFACTURE_REDIS_ADDR", "redis.example.com:6379")
	os.Setenv("ONEFACTURE_REDIS_PASSWORD", "redis-secret")
	os.Setenv("ONEFACTURE_REDIS_DB", "2")
	os.Setenv("ONEFACTURE_REDIS_STREAM", "custom.stream")
	os.Setenv("ONEFACTURE_SIDECAR_URL", "http://sidecar.local:8000")
	os.Setenv("ONEFACTURE_SIDECAR_TIMEOUT", "60s")
	os.Setenv("ONEFACTURE_BOOTSTRAP_API_KEY", "bootstrap-key-123")
	os.Setenv("ONEFACTURE_HASH_PEPPER", "pepper-secret")
	defer clearEnv()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "production", cfg.Env)
	require.Equal(t, ":9000", cfg.HTTP.Addr)
	require.Equal(t, "https://api.example.com", cfg.HTTP.PublicBaseURL)
	require.Equal(t, 1000, cfg.HTTP.RateLimitPerMin)
	require.Equal(t, "postgres://prod:secret@db.example.com/prod", cfg.Database.DSN)
	require.Equal(t, int32(50), cfg.Database.MaxConns)
	require.Equal(t, "migrations/prod", cfg.Database.MigrationsPath)
	require.Equal(t, 10*time.Second, cfg.Database.ConnectTimeout)
	require.False(t, cfg.Database.StatementCache)
	require.Equal(t, "redis.example.com:6379", cfg.Redis.Addr)
	require.Equal(t, "redis-secret", cfg.Redis.Password)
	require.Equal(t, 2, cfg.Redis.DB)
	require.Equal(t, "custom.stream", cfg.Redis.StreamKey)
	require.Equal(t, "http://sidecar.local:8000", cfg.Sidecar.BaseURL)
	require.Equal(t, 60*time.Second, cfg.Sidecar.Timeout)
	require.Equal(t, "bootstrap-key-123", cfg.Auth.BootstrapAPIKey)
	require.Equal(t, "pepper-secret", cfg.Auth.HashPepper)
}

func TestLoadValidation(t *testing.T) {
	clearEnv()

	// In production without ONEFACTURE_HASH_PEPPER, should fail
	os.Setenv("ONEFACTURE_ENV", "production")
	os.Setenv("ONEFACTURE_HASH_PEPPER", "") // explicitly empty
	defer clearEnv()

	cfg, err := Load()
	require.Error(t, err)
	require.Nil(t, cfg)
	require.Contains(t, err.Error(), "ONEFACTURE_HASH_PEPPER")
}

func TestLoadValidationDevelopment(t *testing.T) {
	clearEnv()

	// In development, ONEFACTURE_HASH_PEPPER can be empty
	os.Setenv("ONEFACTURE_ENV", "development")
	defer clearEnv()

	cfg, err := Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "", cfg.Auth.HashPepper)
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		fallback int
		want    int
	}{
		{"valid int", "TEST_INT", "42", 0, 42},
		{"invalid int", "TEST_INT_INVALID", "not-a-number", 99, 99},
		{"missing key", "TEST_INT_MISSING", "", 99, 99},
		{"zero value", "TEST_INT_ZERO", "0", 99, 0},
		{"negative", "TEST_INT_NEG", "-10", 99, -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := envInt(tt.key, tt.fallback)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestEnvInt32(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		fallback int32
		want    int32
	}{
		{"valid int32", "TEST_INT32", "100", 0, 100},
		{"invalid int32", "TEST_INT32_INVALID", "not-a-number", 50, 50},
		{"missing key", "TEST_INT32_MISSING", "", 50, 50},
		{"large value", "TEST_INT32_LARGE", "2147483647", 0, 2147483647}, // max int32
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := envInt32(tt.key, tt.fallback)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback bool
		want     bool
	}{
		{"true", "TEST_BOOL_TRUE", "true", false, true},
		{"false", "TEST_BOOL_FALSE", "false", true, false},
		{"1", "TEST_BOOL_1", "1", false, true},
		{"0", "TEST_BOOL_0", "0", true, false},
		{"t", "TEST_BOOL_T", "t", false, true},
		{"f", "TEST_BOOL_F", "f", true, false},
		{"invalid", "TEST_BOOL_INVALID", "maybe", true, true},
		{"missing key", "TEST_BOOL_MISSING", "", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := envBool(tt.key, tt.fallback)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback time.Duration
		want     time.Duration
	}{
		{"seconds", "TEST_DUR_SEC", "5s", 0, 5 * time.Second},
		{"minutes", "TEST_DUR_MIN", "2m", 0, 2 * time.Minute},
		{"complex", "TEST_DUR_COMPLEX", "1h30m", 0, 1*time.Hour + 30*time.Minute},
		{"invalid", "TEST_DUR_INVALID", "not-a-duration", 10 * time.Second, 10 * time.Second},
		{"missing key", "TEST_DUR_MISSING", "", 30 * time.Second, 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := envDuration(tt.key, tt.fallback)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestEnv(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback string
		want     string
	}{
		{"value set", "TEST_ENV_SET", "custom-value", "default", "custom-value"},
		{"value empty string", "TEST_ENV_EMPTY", "", "default", "default"},
		{"missing key", "TEST_ENV_MISSING", "", "default", "default"},
		{"whitespace preserved", "TEST_ENV_SPACE", "   ", "default", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := env(tt.key, tt.fallback)
			require.Equal(t, tt.want, result)
		})
	}
}

// clearEnv clears all ONEFACTURE_ env vars for testing
func clearEnv() {
	for _, key := range []string{
		"ONEFACTURE_ENV",
		"ONEFACTURE_HTTP_ADDR",
		"ONEFACTURE_PUBLIC_BASE_URL",
		"ONEFACTURE_RATE_LIMIT_PER_MIN",
		"ONEFACTURE_DB_DSN",
		"ONEFACTURE_DB_MAX_CONNS",
		"ONEFACTURE_DB_MIGRATIONS_PATH",
		"ONEFACTURE_DB_CONNECT_TIMEOUT",
		"ONEFACTURE_DB_STATEMENT_CACHE",
		"ONEFACTURE_REDIS_ADDR",
		"ONEFACTURE_REDIS_PASSWORD",
		"ONEFACTURE_REDIS_DB",
		"ONEFACTURE_REDIS_STREAM",
		"ONEFACTURE_SIDECAR_URL",
		"ONEFACTURE_SIDECAR_TIMEOUT",
		"ONEFACTURE_BOOTSTRAP_API_KEY",
		"ONEFACTURE_HASH_PEPPER",
	} {
		os.Unsetenv(key)
	}
}
