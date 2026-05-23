package qonto

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yawo/onefacture/internal/adapters"
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
