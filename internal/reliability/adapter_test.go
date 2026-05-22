package reliability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type flakyAdapter struct {
	calls      int
	failures   int
	alwaysFail bool
}

func (a *flakyAdapter) Name() string { return "flaky" }

func (a *flakyAdapter) Submit(_ context.Context, _ *invoice.Invoice) (*adapters.SubmitResult, error) {
	a.calls++
	if a.alwaysFail || a.calls <= a.failures {
		return nil, errors.New("temporary outage")
	}
	return &adapters.SubmitResult{PARef: "pa_ref", Status: invoice.StatusSubmitted}, nil
}

func (a *flakyAdapter) GetStatus(context.Context, string) (*adapters.LifecycleEvent, error) {
	return nil, adapters.ErrNotImplemented
}

func (a *flakyAdapter) Webhook(context.Context, []byte) (*adapters.WebhookEvent, error) {
	return nil, adapters.ErrNotImplemented
}

func (a *flakyAdapter) HealthCheck(context.Context) error { return nil }

func TestAdapterRetriesSubmitUntilSuccess(t *testing.T) {
	inner := &flakyAdapter{failures: 2}
	wrapped := NewAdapter(inner, NewCircuitBreaker(3, time.Minute), RetryPolicy{MaxAttempts: 3})

	res, err := wrapped.Submit(context.Background(), &invoice.Invoice{})

	require.NoError(t, err)
	require.Equal(t, "pa_ref", res.PARef)
	require.Equal(t, 3, inner.calls)
}

func TestAdapterOpensCircuitAfterFailures(t *testing.T) {
	inner := &flakyAdapter{alwaysFail: true}
	wrapped := NewAdapter(inner, NewCircuitBreaker(2, time.Minute), RetryPolicy{MaxAttempts: 2})

	_, err := wrapped.Submit(context.Background(), &invoice.Invoice{})
	require.Error(t, err)
	_, err = wrapped.Submit(context.Background(), &invoice.Invoice{})

	require.ErrorIs(t, err, ErrCircuitOpen)
	require.Equal(t, 2, inner.calls)
}

func TestAdapterDoesNotRetryNotImplemented(t *testing.T) {
	inner := &notImplementedAdapter{}
	wrapped := NewAdapter(inner, NewCircuitBreaker(3, time.Minute), RetryPolicy{MaxAttempts: 3})

	_, err := wrapped.Submit(context.Background(), &invoice.Invoice{})

	require.ErrorIs(t, err, adapters.ErrNotImplemented)
	require.Equal(t, 1, inner.calls)
}

type notImplementedAdapter struct {
	flakyAdapter
}

func (a *notImplementedAdapter) Submit(context.Context, *invoice.Invoice) (*adapters.SubmitResult, error) {
	a.calls++
	return nil, adapters.ErrNotImplemented
}
