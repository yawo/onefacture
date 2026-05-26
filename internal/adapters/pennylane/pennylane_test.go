package pennylane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestNew(t *testing.T) {
	adapter := New()
	require.NotNil(t, adapter)
}

func TestNewConfiguresSandboxClientFromEnv(t *testing.T) {
	t.Setenv("ONEFACTURE_PENNYLANE_BASE_URL", "https://pennylane.example.test")
	t.Setenv("ONEFACTURE_PENNYLANE_API_TOKEN", "pennylane-token")
	t.Setenv("ONEFACTURE_PENNYLANE_SUBMIT_PATH", "/custom-submit")
	t.Setenv("ONEFACTURE_PENNYLANE_STATUS_PATH", "/custom-status/{pa_ref}")
	t.Setenv("ONEFACTURE_PENNYLANE_WEBHOOK_KEY", "pennylane-webhook")

	adapter := New()

	require.Equal(t, "pennylane", adapter.client.Name)
	require.Equal(t, "https://pennylane.example.test", adapter.client.BaseURL)
	require.Equal(t, "/custom-submit", adapter.client.SubmitPath)
	require.Equal(t, "/custom-status/{pa_ref}", adapter.client.StatusPath)
	require.Equal(t, "pennylane-webhook", adapter.client.WebhookKey)
	require.Equal(t, "Bearer", adapter.client.Auth.Scheme)
	require.Equal(t, "pennylane-token", adapter.client.Auth.Token)
}

func TestName(t *testing.T) {
	adapter := New()
	require.Equal(t, "pennylane", adapter.Name())
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

func TestPennylaneIntegrationSubmitAndGetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		switch {
		case r.URL.Path == "/invoices" && r.Method == "POST":
			var inv invoice.Invoice
			require.NoError(t, json.NewDecoder(r.Body).Decode(&inv))
			require.Equal(t, "test-inv", inv.Number)
			_ = json.NewEncoder(w).Encode(adapters.SubmitResult{PARef: "penny-123", Status: invoice.StatusSubmitted})
		case strings.HasSuffix(r.URL.Path, "/status"):
			_ = json.NewEncoder(w).Encode(adapters.LifecycleEvent{PARef: "penny-123", Status: invoice.StatusAccepted})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := sandbox.Client{Name: "pennylane", BaseURL: server.URL, SubmitPath: "/invoices", StatusPath: "/invoices/{pa_ref}/status", Auth: sandbox.Auth{Token: "token"}, HTTP: server.Client()}
	a := &Adapter{client: c}
	res, err := a.Submit(context.Background(), &invoice.Invoice{Number: "test-inv"})
	require.NoError(t, err)
	require.Equal(t, "penny-123", res.PARef)
	require.Equal(t, invoice.StatusSubmitted, res.Status)
	ev, err := a.GetStatus(context.Background(), "penny-123")
	require.NoError(t, err)
	require.Equal(t, "penny-123", ev.PARef)
	require.Equal(t, invoice.StatusAccepted, ev.Status)
}
