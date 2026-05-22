package docaposte

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestNew(t *testing.T) {
	adapter := New()
	require.NotNil(t, adapter)
}

func TestNewConfiguresSandboxClientFromEnv(t *testing.T) {
	t.Setenv("ONEFACTURE_DOCAPOSTE_BASE_URL", "https://docaposte.example.test")
	t.Setenv("ONEFACTURE_DOCAPOSTE_API_TOKEN", "docaposte-token")
	t.Setenv("ONEFACTURE_DOCAPOSTE_SUBMIT_PATH", "/custom-submit")
	t.Setenv("ONEFACTURE_DOCAPOSTE_STATUS_PATH", "/custom-status/{pa_ref}")
	t.Setenv("ONEFACTURE_DOCAPOSTE_WEBHOOK_KEY", "docaposte-webhook")

	adapter := New()

	require.Equal(t, "docaposte", adapter.client.Name)
	require.Equal(t, "https://docaposte.example.test", adapter.client.BaseURL)
	require.Equal(t, "/custom-submit", adapter.client.SubmitPath)
	require.Equal(t, "/custom-status/{pa_ref}", adapter.client.StatusPath)
	require.Equal(t, "docaposte-webhook", adapter.client.WebhookKey)
	require.Equal(t, "Bearer", adapter.client.Auth.Scheme)
	require.Equal(t, "docaposte-token", adapter.client.Auth.Token)
}

func TestName(t *testing.T) {
	adapter := New()
	require.Equal(t, "docaposte", adapter.Name())
}

func TestHealthCheck(t *testing.T) {
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
