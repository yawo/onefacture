package chorus

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestNew(t *testing.T) {
	// Clear env vars
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
	os.Unsetenv("ONEFACTURE_CHORUS_BASE_URL")

	adapter := New()
	require.NotNil(t, adapter)
	require.Equal(t, "", adapter.clientID)
	require.Equal(t, "", adapter.clientSecret)
	require.Equal(t, "https://sandbox-api.piste.gouv.fr/cpro", adapter.baseURL)
}

func TestNewWithEnv(t *testing.T) {
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_ID", "test-client-id")
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_SECRET", "test-client-secret")
	os.Setenv("ONEFACTURE_CHORUS_BASE_URL", "https://custom-url.example.com")
	defer func() {
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
		os.Unsetenv("ONEFACTURE_CHORUS_BASE_URL")
	}()

	adapter := New()
	require.Equal(t, "test-client-id", adapter.clientID)
	require.Equal(t, "test-client-secret", adapter.clientSecret)
	require.Equal(t, "https://custom-url.example.com", adapter.baseURL)
}

func TestName(t *testing.T) {
	adapter := New()
	require.Equal(t, "chorus", adapter.Name())
}

func TestHealthCheckWithoutCredentials(t *testing.T) {
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")

	adapter := New()
	err := adapter.HealthCheck(context.Background())
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestHealthCheckWithCredentials(t *testing.T) {
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_ID", "test-id")
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_SECRET", "test-secret")
	defer func() {
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
	}()

	adapter := New()
	err := adapter.HealthCheck(context.Background())
	require.NoError(t, err)
}

func TestHealthCheckPartialCredentials(t *testing.T) {
	os.Setenv("ONEFACTURE_CHORUS_CLIENT_ID", "test-id")
	os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_SECRET")
	defer func() {
		os.Unsetenv("ONEFACTURE_CHORUS_CLIENT_ID")
	}()

	adapter := New()
	err := adapter.HealthCheck(context.Background())
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestSubmit(t *testing.T) {
	adapter := New()
	inv := &invoice.Invoice{}
	result, err := adapter.Submit(context.Background(), inv)
	require.Nil(t, result)
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestGetStatus(t *testing.T) {
	adapter := New()
	event, err := adapter.GetStatus(context.Background(), "test-ref")
	require.Nil(t, event)
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestWebhook(t *testing.T) {
	adapter := New()
	event, err := adapter.Webhook(context.Background(), []byte("{}"))
	require.Nil(t, event)
	require.Equal(t, adapters.ErrNotImplemented, err)
}

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		fallback string
		expected string
	}{
		{"value set", "TEST_KEY_SET", "custom-value", "default", "custom-value"},
		{"value not set", "TEST_KEY_UNSET", "", "default", "default"},
		{"empty value", "TEST_KEY_EMPTY", "", "fallback", "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.value != "" {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}
			result := envOr(tt.key, tt.fallback)
			require.Equal(t, tt.expected, result)
		})
	}
}
