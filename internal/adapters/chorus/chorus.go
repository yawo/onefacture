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
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Adapter struct {
	clientID     string
	clientSecret string
	baseURL      string
	client       sandbox.Client
}

func New() *Adapter {
	a := &Adapter{
		clientID:     os.Getenv("ONEFACTURE_CHORUS_CLIENT_ID"),
		clientSecret: os.Getenv("ONEFACTURE_CHORUS_CLIENT_SECRET"),
		baseURL:      envOr("ONEFACTURE_CHORUS_BASE_URL", "https://sandbox-api.piste.gouv.fr/cpro"),
	}
	a.client = sandbox.Client{
		Name:       "chorus",
		BaseURL:    a.baseURL,
		SubmitPath: envOr("ONEFACTURE_CHORUS_SUBMIT_PATH", "/invoices"),
		StatusPath: envOr("ONEFACTURE_CHORUS_STATUS_PATH", "/invoices/{pa_ref}/status"),
		WebhookKey: os.Getenv("ONEFACTURE_CHORUS_WEBHOOK_KEY"),
		Auth: sandbox.Auth{
			Scheme:       "Bearer",
			Token:        os.Getenv("ONEFACTURE_CHORUS_ACCESS_TOKEN"),
			TokenURL:     envOr("ONEFACTURE_CHORUS_TOKEN_URL", "https://sandbox-oauth.piste.gouv.fr/api/oauth/token"),
			ClientID:     a.clientID,
			ClientSecret: a.clientSecret,
			Scope:        os.Getenv("ONEFACTURE_CHORUS_SCOPE"),
		},
	}
	return a
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (a *Adapter) Name() string { return "chorus" }

func (a *Adapter) HealthCheck(ctx context.Context) error {
	return a.client.HealthCheck(ctx)
}

func (a *Adapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	return a.client.Submit(ctx, inv)
}

func (a *Adapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	return a.client.GetStatus(ctx, paRef)
}

func (a *Adapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	return a.client.Webhook(ctx, payload)
}
