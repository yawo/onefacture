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
