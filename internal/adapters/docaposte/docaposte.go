// Package docaposte is the Docaposte (SERES) PDP adapter (stub).
package docaposte

import (
	"context"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/core/invoice"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string                        { return "docaposte" }
func (a *Adapter) HealthCheck(_ context.Context) error { return adapters.ErrNotImplemented }
func (a *Adapter) Submit(_ context.Context, _ *invoice.Invoice) (*adapters.SubmitResult, error) {
	return nil, adapters.ErrNotImplemented
}
func (a *Adapter) GetStatus(_ context.Context, _ string) (*adapters.LifecycleEvent, error) {
	return nil, adapters.ErrNotImplemented
}
func (a *Adapter) Webhook(_ context.Context, _ []byte) (*adapters.WebhookEvent, error) {
	return nil, adapters.ErrNotImplemented
}
