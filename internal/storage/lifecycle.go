package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yawo/onefacture/internal/core/invoice"
)

type LifecycleEvent struct {
	ID         uuid.UUID
	InvoiceID  uuid.UUID
	FromStatus invoice.Status
	ToStatus   invoice.Status
	PACode     string
	PAMessage  string
	Payload    map[string]any
	OccurredAt time.Time
}

type LifecycleRepo struct{ pool *pgxpool.Pool }

func (r *LifecycleRepo) Record(ctx context.Context, orgID, invoiceID uuid.UUID, ev LifecycleEvent) error {
	payload, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	const q = `INSERT INTO lifecycle_events
(invoice_id, organization_id, from_status, to_status, pa_code, pa_message, payload)
VALUES ($1,$2,NULLIF($3,'')::invoice_status,$4,NULLIF($5,''),NULLIF($6,''),$7)`
	_, err = r.pool.Exec(ctx, q, invoiceID, orgID, string(ev.FromStatus), ev.ToStatus, ev.PACode, ev.PAMessage, payload)
	if err != nil {
		return fmt.Errorf("insert lifecycle event: %w", err)
	}
	return nil
}

func (r *LifecycleRepo) List(ctx context.Context, orgID, invoiceID uuid.UUID) ([]LifecycleEvent, error) {
	const q = `SELECT id, invoice_id, COALESCE(from_status::text,''), to_status, COALESCE(pa_code,''), COALESCE(pa_message,''), COALESCE(payload, '{}'::jsonb), occurred_at
FROM lifecycle_events WHERE organization_id = $1 AND invoice_id = $2 ORDER BY occurred_at ASC`
	rows, err := r.pool.Query(ctx, q, orgID, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	out := []LifecycleEvent{}
	for rows.Next() {
		ev := LifecycleEvent{}
		var from, to string
		var payload []byte
		if err := rows.Scan(&ev.ID, &ev.InvoiceID, &from, &to, &ev.PACode, &ev.PAMessage, &payload, &ev.OccurredAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ev.FromStatus = invoice.Status(from)
		ev.ToStatus = invoice.Status(to)
		_ = json.Unmarshal(payload, &ev.Payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}
