package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestMockLifecycle(t *testing.T) {
	a := New()
	inv := &invoice.Invoice{Number: "INV-1"}
	res, err := a.Submit(context.Background(), inv)
	require.NoError(t, err)
	require.NotEmpty(t, res.PARef)
	require.Equal(t, invoice.StatusSubmitted, res.Status)

	ev, err := a.GetStatus(context.Background(), res.PARef)
	require.NoError(t, err)
	require.Equal(t, invoice.StatusReceived, ev.Status)

	ev, err = a.GetStatus(context.Background(), res.PARef)
	require.NoError(t, err)
	require.Equal(t, invoice.StatusAccepted, ev.Status)
}

func TestMockName(t *testing.T) {
a := New()
require.Equal(t, "mock", a.Name())
}

func TestMockHealthCheck(t *testing.T) {
a := New()
err := a.HealthCheck(context.Background())
require.NoError(t, err)
}

func TestMockGetStatusUnknownRef(t *testing.T) {
a := New()
_, err := a.GetStatus(context.Background(), "unknown_ref")
require.Error(t, err)
require.Contains(t, err.Error(), "unknown pa_ref")
}

func TestMockWebhookValidPayload(t *testing.T) {
a := New()
payload := []byte(`{"pa_ref":"test_ref","status":"accepted","event_type":"invoice.paid"}`)
ev, err := a.Webhook(context.Background(), payload)
require.NoError(t, err)
require.Equal(t, "test_ref", ev.PARef)
require.Equal(t, invoice.Status("accepted"), ev.Status)
require.Equal(t, "invoice.paid", ev.EventType)
}

func TestMockWebhookInvalidJSON(t *testing.T) {
a := New()
payload := []byte(`{invalid json}`)
_, err := a.Webhook(context.Background(), payload)
require.Error(t, err)
}

func TestMockWebhookDefaultEventType(t *testing.T) {
a := New()
payload := []byte(`{"pa_ref":"test_ref","status":"accepted"}`)
ev, err := a.Webhook(context.Background(), payload)
require.NoError(t, err)
require.Equal(t, "invoice.updated", ev.EventType)
}

func TestMockMultipleSubmissions(t *testing.T) {
a := New()
inv1 := &invoice.Invoice{Number: "INV-1"}
inv2 := &invoice.Invoice{Number: "INV-2"}

res1, err := a.Submit(context.Background(), inv1)
require.NoError(t, err)

res2, err := a.Submit(context.Background(), inv2)
require.NoError(t, err)

require.NotEqual(t, res1.PARef, res2.PARef)

// Both should track independently
ev1, err := a.GetStatus(context.Background(), res1.PARef)
require.NoError(t, err)
require.Equal(t, invoice.StatusReceived, ev1.Status)
}
