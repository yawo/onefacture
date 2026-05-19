package adapters

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/core/invoice"
)

func TestErrNotImplementedString(t *testing.T) {
	err := ErrNotImplemented
	require.Equal(t, "adapter: not implemented", err.Error())
	require.Equal(t, "adapter: not implemented", string(err))
}

func TestErrNotImplementedType(t *testing.T) {
	var err error = ErrNotImplemented
	require.NotNil(t, err)
	require.Error(t, err)
}

func TestSubmitResultStruct(t *testing.T) {
	// Ensure struct is properly defined
	result := &SubmitResult{
		PARef:  "test-ref",
		Status: invoice.StatusSubmitted,
	}
	require.Equal(t, "test-ref", result.PARef)
	require.Equal(t, invoice.StatusSubmitted, result.Status)
}

func TestLifecycleEventStruct(t *testing.T) {
	// Ensure struct is properly defined
	event := &LifecycleEvent{
		PARef:     "test-ref",
		Status:    invoice.StatusValidated,
		PACode:    "PA-123",
		PAMessage: "Validated successfully",
	}
	require.Equal(t, "test-ref", event.PARef)
	require.Equal(t, invoice.StatusValidated, event.Status)
	require.Equal(t, "PA-123", event.PACode)
	require.Equal(t, "Validated successfully", event.PAMessage)
}

func TestWebhookEventStruct(t *testing.T) {
	// Ensure struct is properly defined
	event := &WebhookEvent{
		EventType: "invoice.validated",
		PARef:     "test-ref",
		Status:    invoice.StatusValidated,
		Payload: map[string]any{
			"test": "value",
		},
	}
	require.Equal(t, "invoice.validated", event.EventType)
	require.Equal(t, "test-ref", event.PARef)
	require.Equal(t, invoice.StatusValidated, event.Status)
	require.NotNil(t, event.Payload)
	require.Equal(t, "value", event.Payload["test"])
}
