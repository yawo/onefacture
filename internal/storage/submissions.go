package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yawo/onefacture/internal/metrics"
)

type SubmissionDLQEntry struct {
	ID             uuid.UUID      `json:"id"`
	OrganizationID uuid.UUID      `json:"organization_id"`
	InvoiceID      uuid.UUID      `json:"invoice_id"`
	PAID           string         `json:"pa_id"`
	Error          string         `json:"error"`
	Payload        map[string]any `json:"payload"`
	Status         string         `json:"status"`
	CreatedAt      time.Time      `json:"created_at"`
	ReplayedAt     *time.Time     `json:"replayed_at,omitempty"`
}

type SubmissionRepo struct{ pool *pgxpool.Pool }

func (r *SubmissionRepo) EnqueueDLQ(ctx context.Context, orgID, invoiceID uuid.UUID, paID, errMsg string, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal submission dlq payload: %w", err)
	}
	const q = `INSERT INTO submission_dlq (organization_id, invoice_id, pa_id, error, payload)
VALUES ($1,$2,$3,$4,$5)`
	if _, err := r.pool.Exec(ctx, q, orgID, invoiceID, paID, errMsg, raw); err != nil {
		return fmt.Errorf("insert submission dlq: %w", err)
	}
	metrics.DLQDepth.Add(1)
	metrics.DLQEnqueuedTotal.WithLabelValues(paID).Inc()
	return nil
}

func (r *SubmissionRepo) ListDLQ(ctx context.Context, orgID uuid.UUID, limit int) ([]SubmissionDLQEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `SELECT id, organization_id, invoice_id, pa_id, error, payload, status, created_at, replayed_at
FROM submission_dlq WHERE organization_id = $1
ORDER BY created_at DESC LIMIT $2`
	rows, err := r.pool.Query(ctx, q, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list submission dlq: %w", err)
	}
	defer rows.Close()
	out := []SubmissionDLQEntry{}
	for rows.Next() {
		entry := SubmissionDLQEntry{}
		var payload []byte
		if err := rows.Scan(&entry.ID, &entry.OrganizationID, &entry.InvoiceID, &entry.PAID, &entry.Error, &payload, &entry.Status, &entry.CreatedAt, &entry.ReplayedAt); err != nil {
			return nil, fmt.Errorf("scan submission dlq: %w", err)
		}
		_ = json.Unmarshal(payload, &entry.Payload)
		out = append(out, entry)
	}
	return out, rows.Err()
}

func (r *SubmissionRepo) GetDLQ(ctx context.Context, orgID, id uuid.UUID) (*SubmissionDLQEntry, error) {
	const q = `SELECT id, organization_id, invoice_id, pa_id, error, payload, status, created_at, replayed_at
FROM submission_dlq WHERE organization_id = $1 AND id = $2`
	entry := &SubmissionDLQEntry{}
	var payload []byte
	err := r.pool.QueryRow(ctx, q, orgID, id).Scan(&entry.ID, &entry.OrganizationID, &entry.InvoiceID, &entry.PAID, &entry.Error, &payload, &entry.Status, &entry.CreatedAt, &entry.ReplayedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get submission dlq: %w", err)
	}
	_ = json.Unmarshal(payload, &entry.Payload)
	return entry, nil
}

func (r *SubmissionRepo) MarkReplayed(ctx context.Context, orgID, id uuid.UUID) error {
	const q = `UPDATE submission_dlq SET status = 'REPLAYED', replayed_at = now()
WHERE organization_id = $1 AND id = $2`
	tag, err := r.pool.Exec(ctx, q, orgID, id)
	if err != nil {
		return fmt.Errorf("mark submission dlq replayed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	metrics.DLQDepth.Add(-1)
	return nil
}
