// Package pennylane is the Pennylane PDP adapter.
package pennylane

import (
	"context"
	"os"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/sandbox"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Adapter struct{ client sandbox.Client }

func New() *Adapter {
	return &Adapter{client: sandbox.Client{
		Name:       "pennylane",
		BaseURL:    os.Getenv("ONEFACTURE_PENNYLANE_BASE_URL"),
		SubmitPath: envOr("ONEFACTURE_PENNYLANE_SUBMIT_PATH", "/invoices"),
		StatusPath: envOr("ONEFACTURE_PENNYLANE_STATUS_PATH", "/invoices/{pa_ref}/status"),
		WebhookKey: os.Getenv("ONEFACTURE_PENNYLANE_WEBHOOK_KEY"),
		Auth:       sandbox.Auth{Scheme: "Bearer", Token: os.Getenv("ONEFACTURE_PENNYLANE_API_TOKEN")},
	}}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func (a *Adapter) Name() string { return "pennylane" }

func (a *Adapter) HealthCheck(ctx context.Context) error { return a.client.HealthCheck(ctx) }

func (a *Adapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	return a.client.Submit(ctx, inv)
}

func (a *Adapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	return a.client.GetStatus(ctx, paRef)
}

func (a *Adapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	return a.client.Webhook(ctx, payload)
}
