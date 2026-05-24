package qonto

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestQontoAdapterName(t *testing.T) {
	a := New()
	require.Equal(t, "qonto", a.Name())
}

func TestQontoAdapterImplementsInterface(t *testing.T) {
	var _ adapters.PAAdapter = New()
}

func TestQontoHealthCheckWithoutCreds(t *testing.T) {
	a := New()
	err := a.HealthCheck(context.Background())
	_ = err
}

func TestQontoErrorMapping(t *testing.T) {
	a := New()

	rawErr := &adapters.PAError{
		Platform:  "sandbox",
		Operation: "submit",
		Code:      "QON-101",
		Message:   "invalid iban",
	}

	mapped := a.mapQontoError("submit", rawErr)

	var paErr *adapters.PAError
	require.True(t, errors.As(mapped, &paErr))
	require.Equal(t, "qonto", paErr.Platform)
	require.Equal(t, "QONTO_INVALID_IBAN", paErr.Code)
	require.Contains(t, paErr.Remediation, "IBAN")
}

func TestQontoNormalizeLifecycleStatus(t *testing.T) {
	require.Equal(t, "ACCEPTED", NormalizeLifecycleStatus("accepted"))
	require.Equal(t, "REJECTED", NormalizeLifecycleStatus("rejected"))
	require.Equal(t, "SUBMITTED", NormalizeLifecycleStatus("pending"))
}

func TestQontoIntegrationNormalizeLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/invoices" {
			_ = json.NewEncoder(w).Encode(adapters.SubmitResult{PARef: "pa-qon-1", Status: "pending"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/status") {
			_ = json.NewEncoder(w).Encode(adapters.LifecycleEvent{PARef: "pa-qon-1", Status: "accepted"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := sandbox.Client{Name: "qonto", BaseURL: server.URL, SubmitPath: "/invoices", StatusPath: "/invoices/{pa_ref}/status", Auth: sandbox.Auth{Token: "t"}, HTTP: server.Client()}
	a := &Adapter{client: c}
	res, err := a.Submit(context.Background(), &invoice.Invoice{Number: "i1"})
	require.NoError(t, err)
	require.Equal(t, invoice.StatusSubmitted, res.Status)
	ev, err := a.GetStatus(context.Background(), "pa-qon-1")
	require.NoError(t, err)
	require.Equal(t, invoice.StatusAccepted, ev.Status)
}
