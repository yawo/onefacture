// Package workers contains the background workers that drive the async pipeline.
package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/metrics"
	"github.com/yawo/onefacture/internal/storage"
)

// StatusPoller periodically polls PA adapters for status updates on invoices
// that are still in flight (SUBMITTED or RECEIVED).
type StatusPoller struct {
	logger   *slog.Logger
	store    *storage.Store
	registry *registry.Registry
	bus      *events.Bus
	interval time.Duration
}

func NewStatusPoller(logger *slog.Logger, store *storage.Store, reg *registry.Registry, bus *events.Bus) *StatusPoller {
	return &StatusPoller{
		logger: logger, store: store, registry: reg, bus: bus,
		interval: 30 * time.Second,
	}
}

func (p *StatusPoller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics.StatusPollsTotal.Inc()
			p.tick(ctx)
		}
	}
}

func (p *StatusPoller) tick(ctx context.Context) {
	// Scoped to keep the example small: a production version paginates.
	const q = `SELECT id, organization_id, pa_id, pa_ref, status::text FROM invoices
WHERE status IN ('SUBMITTED','RECEIVED') AND pa_ref IS NOT NULL LIMIT 100`
	rows, err := p.store.Pool().Query(ctx, q)
	if err != nil {
		p.logger.Warn("status poll query", "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, orgID uuid.UUID
		var paID, paRef, status string
		if err := rows.Scan(&id, &orgID, &paID, &paRef, &status); err != nil {
			p.logger.Warn("scan", "err", err)
			continue
		}
		adapter, err := p.registry.Get(paID)
		if err != nil {
			continue
		}
		ev, err := adapter.GetStatus(ctx, paRef)
		if err != nil {
			p.logger.Debug("get status", "err", err, "pa_ref", paRef)
			continue
		}
		if ev.Status == "" || string(ev.Status) == status {
			continue
		}
		if err := p.store.Invoices.UpdateStatus(ctx, orgID, id, ev.Status); err != nil {
			p.logger.Warn("update status", "err", err)
			continue
		}
		if ev.Status == invoice.StatusRejected {
			_ = p.store.Invoices.SetLastRejection(ctx, orgID, id, invoice.Rejection{
				Code:       ev.PACode,
				Message:    ev.PAMessage,
				OccurredAt: ev.OccurredAt,
			})
		}
		_ = p.store.Lifecycle.Record(ctx, orgID, id, storage.LifecycleEvent{
			FromStatus: invoice.Status(status), ToStatus: ev.Status, PACode: ev.PACode, PAMessage: ev.PAMessage,
		})
		_ = p.bus.Publish(ctx, events.Event{
			Type: "invoice." + string(ev.Status), OrganizationID: orgID.String(), InvoiceID: id.String(),
		})
	}
}
