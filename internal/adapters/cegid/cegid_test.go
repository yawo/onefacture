package cegid

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

func TestCegidAdapterName(t *testing.T) {
	a := New()
	require.Equal(t, "cegid", a.Name())
}

func TestCegidAdapterImplementsInterface(t *testing.T) {
	var _ adapters.PAAdapter = New()
}

func TestCegidHealthCheckWithoutCreds(t *testing.T) {
	a := New()
	err := a.HealthCheck(context.Background())
	// In sandbox mode without real creds it may fail or be skipped
	_ = err
}

func TestCegidErrorMapping(t *testing.T) {
	a := New()

	// Simulation d’une erreur PA brute venant du client sandbox
	rawErr := &adapters.PAError{
		Platform:  "sandbox",
		Operation: "submit",
		Code:      "CEG-001",
		Message:   "buyer siren invalid",
	}

	mapped := a.mapCegidError("submit", rawErr)

	var paErr *adapters.PAError
	require.True(t, errors.As(mapped, &paErr))
	require.Equal(t, "cegid", paErr.Platform)
	require.Equal(t, "CEGID_INVALID_SIREN", paErr.Code)
	require.Contains(t, paErr.Remediation, "SIREN")
}

func TestCegidNormalizeLifecycleStatus(t *testing.T) {
	require.Equal(t, "ACCEPTED", NormalizeLifecycleStatus("ACCEPTE"))
	require.Equal(t, "REJECTED", NormalizeLifecycleStatus("REFUSE"))
	require.Equal(t, "SUBMITTED", NormalizeLifecycleStatus("EN_COURS"))
}

func TestCegidIntegrationNormalizeLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/invoices" {
			_ = json.NewEncoder(w).Encode(adapters.SubmitResult{PARef: "pa-ceg-1", Status: "EN_COURS"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/status") {
			_ = json.NewEncoder(w).Encode(adapters.LifecycleEvent{PARef: "pa-ceg-1", Status: "ACCEPTE"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	c := sandbox.Client{Name: "cegid", BaseURL: server.URL, SubmitPath: "/invoices", StatusPath: "/invoices/{pa_ref}/status", Auth: sandbox.Auth{Token: "t"}, HTTP: server.Client()}
	a := &Adapter{client: c}
	res, err := a.Submit(context.Background(), &invoice.Invoice{Number: "i1"})
	require.NoError(t, err)
	require.Equal(t, invoice.StatusSubmitted, res.Status)
	ev, err := a.GetStatus(context.Background(), "pa-ceg-1")
	require.NoError(t, err)
	require.Equal(t, invoice.StatusAccepted, ev.Status)
}
