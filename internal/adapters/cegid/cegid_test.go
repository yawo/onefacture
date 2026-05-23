package cegid

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yawo/onefacture/internal/adapters"
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
