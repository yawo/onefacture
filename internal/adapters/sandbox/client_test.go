package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestClientSubmitAndStatusRoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/invoices":
			_ = json.NewEncoder(w).Encode(adapters.SubmitResult{PARef: "pa-123", Status: invoice.StatusSubmitted})
		case "/invoices/pa-123/status":
			_ = json.NewEncoder(w).Encode(adapters.LifecycleEvent{PARef: "pa-123", Status: invoice.StatusAccepted})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := Client{
		Name:       "sandbox",
		BaseURL:    server.URL,
		SubmitPath: "/invoices",
		StatusPath: "/invoices/{pa_ref}/status",
		Auth:       Auth{Token: "token"},
		HTTP:       server.Client(),
	}

	submit, err := client.Submit(context.Background(), &invoice.Invoice{Number: "INV-1"})
	require.NoError(t, err)
	require.Equal(t, "pa-123", submit.PARef)

	status, err := client.GetStatus(context.Background(), "pa-123")
	require.NoError(t, err)
	require.Equal(t, invoice.StatusAccepted, status.Status)
}

func TestClientUsesOAuthClientCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			require.Equal(t, http.MethodPost, r.Method)
			require.NoError(t, r.ParseForm())
			require.Equal(t, "client_credentials", r.Form.Get("grant_type"))
			require.Equal(t, "client-id", r.Form.Get("client_id"))
			require.Equal(t, "client-secret", r.Form.Get("client_secret"))
			require.Equal(t, "piste.scope", r.Form.Get("scope"))
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "oauth-token", "token_type": "Bearer"})
		case "/invoices":
			require.Equal(t, "Bearer oauth-token", r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(adapters.SubmitResult{PARef: "pa-oauth", Status: invoice.StatusSubmitted})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := Client{
		Name:       "chorus",
		BaseURL:    server.URL,
		SubmitPath: "/invoices",
		Auth: Auth{
			TokenURL:     server.URL + "/oauth/token",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			Scope:        "piste.scope",
		},
		HTTP: server.Client(),
	}

	submit, err := client.Submit(context.Background(), &invoice.Invoice{Number: "INV-1"})

	require.NoError(t, err)
	require.Equal(t, "pa-oauth", submit.PARef)
}

func TestClientMapsPAErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":      "BR-SIREN",
			"message":   "buyer siren invalid",
			"retryable": false,
		})
	}))
	defer server.Close()

	client := Client{
		Name:       "sandbox",
		BaseURL:    server.URL,
		SubmitPath: "/invoices",
		Auth:       Auth{Token: "token"},
		HTTP:       server.Client(),
	}

	result, err := client.Submit(context.Background(), &invoice.Invoice{Number: "INV-1"})

	require.Nil(t, result)
	var paErr *adapters.PAError
	require.True(t, errors.As(err, &paErr))
	require.Equal(t, "sandbox", paErr.Platform)
	require.Equal(t, "submit", paErr.Operation)
	require.Equal(t, http.StatusBadRequest, paErr.StatusCode)
	require.Equal(t, "BR-SIREN", paErr.Code)
	require.Equal(t, "buyer siren invalid", paErr.Message)
	require.False(t, paErr.Retryable)
}

func TestClientWithoutTokenReturnsNotImplemented(t *testing.T) {
	client := Client{Name: "sandbox", BaseURL: "https://sandbox.example", SubmitPath: "/invoices"}

	_, err := client.Submit(context.Background(), &invoice.Invoice{})

	require.ErrorIs(t, err, adapters.ErrNotImplemented)
}

func TestClientWebhookDecode(t *testing.T) {
	client := Client{Name: "sandbox", WebhookKey: "secret"}

	ev, err := client.Webhook(context.Background(), []byte(`{"event_type":"invoice.accepted","pa_ref":"pa-123","status":"ACCEPTED"}`))

	require.NoError(t, err)
	require.Equal(t, "invoice.accepted", ev.EventType)
	require.Equal(t, invoice.StatusAccepted, ev.Status)
}
