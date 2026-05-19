// Package chorus is the Chorus Pro / PPF adapter (PISTE OAuth2).
//
// The real implementation lives in chorus_real.go (build tag: piste). Without
// that tag, the adapter operates in stub mode: every call returns
// ErrNotImplemented so callers can surface a graceful 501.
package chorus

import (
	"context"
	"os"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Adapter struct {
	clientID     string
	clientSecret string
	baseURL      string
}

func New() *Adapter {
	return &Adapter{
		clientID:     os.Getenv("ONEFACTURE_CHORUS_CLIENT_ID"),
		clientSecret: os.Getenv("ONEFACTURE_CHORUS_CLIENT_SECRET"),
		baseURL:      envOr("ONEFACTURE_CHORUS_BASE_URL", "https://sandbox-api.piste.gouv.fr/cpro"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (a *Adapter) Name() string { return "chorus" }

func (a *Adapter) HealthCheck(_ context.Context) error {
	if a.clientID == "" || a.clientSecret == "" {
		return adapters.ErrNotImplemented
	}
	return nil
}

func (a *Adapter) Submit(_ context.Context, _ *invoice.Invoice) (*adapters.SubmitResult, error) {
	return nil, adapters.ErrNotImplemented
}

func (a *Adapter) GetStatus(_ context.Context, _ string) (*adapters.LifecycleEvent, error) {
	return nil, adapters.ErrNotImplemented
}

func (a *Adapter) Webhook(_ context.Context, _ []byte) (*adapters.WebhookEvent, error) {
	return nil, adapters.ErrNotImplemented
}
