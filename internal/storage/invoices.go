package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yawo/onefacture/internal/core/invoice"
)

// Direction of an invoice as stored.
type Direction string

const (
	DirectionOutbound Direction = "OUTBOUND"
	DirectionInbound  Direction = "INBOUND"
)

type InvoiceRow struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Direction      Direction
	Invoice        *invoice.Invoice
}

type InvoiceRepo struct{ pool *pgxpool.Pool }

// Create persists an invoice and returns the assigned ID.
func (r *InvoiceRepo) Create(ctx context.Context, orgID uuid.UUID, dir Direction, inv *invoice.Invoice) (uuid.UUID, error) {
	id := uuid.New()
	payload, err := json.Marshal(inv)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal invoice: %w", err)
	}
	const q = `
INSERT INTO invoices (
    id, organization_id, direction, status, profile, type_code, number, currency,
    issue_date, due_date, seller_siren, buyer_siren, pa_id, pa_ref,
    payload, raw_xml, raw_pdf
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),NULLIF($14,''),$15,$16,$17)
RETURNING id, created_at, updated_at`
	var due any
	if inv.DueDate != nil {
		due = *inv.DueDate
	}
	var created, updated time.Time
	err = r.pool.QueryRow(ctx, q,
		id, orgID, dir, inv.Status, inv.Profile, inv.TypeCode, inv.Number, inv.Currency,
		inv.IssueDate, due, inv.Seller.SIREN, inv.Buyer.SIREN, inv.PAID, inv.PARef,
		payload, inv.RawXML, inv.RawPDF,
	).Scan(&inv.ID, &created, &updated)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert invoice: %w", err)
	}
	inv.OrganizationID = orgID.String()
	inv.CreatedAt = created
	inv.UpdatedAt = updated
	return id, nil
}

// Get fetches a single invoice scoped to the organization.
func (r *InvoiceRepo) Get(ctx context.Context, orgID, id uuid.UUID) (*invoice.Invoice, error) {
	const q = `SELECT payload, status, created_at, updated_at FROM invoices
WHERE id = $1 AND organization_id = $2`
	var payload []byte
	var status invoice.Status
	var created, updated time.Time
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(&payload, &status, &created, &updated)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}
	inv := &invoice.Invoice{}
	if err := json.Unmarshal(payload, inv); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	inv.Status = status
	inv.ID = id.String()
	inv.OrganizationID = orgID.String()
	inv.CreatedAt = created
	inv.UpdatedAt = updated
	return inv, nil
}

// UpdateStatus changes the invoice status (no state-machine check; callers enforce).
func (r *InvoiceRepo) UpdateStatus(ctx context.Context, orgID, id uuid.UUID, status invoice.Status) error {
	const q = `UPDATE invoices SET status = $1, updated_at = now() WHERE id = $2 AND organization_id = $3`
	tag, err := r.pool.Exec(ctx, q, status, id, orgID)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// List returns invoices for an organization, paginated and optionally filtered.
type ListFilter struct {
	Direction Direction
	Status    invoice.Status
	Limit     int
	Offset    int
}

func (r *InvoiceRepo) List(ctx context.Context, orgID uuid.UUID, f ListFilter) ([]*invoice.Invoice, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	q := `SELECT id, payload, status, created_at, updated_at FROM invoices WHERE organization_id = $1`
	args := []any{orgID}
	if f.Direction != "" {
		args = append(args, f.Direction)
		q += fmt.Sprintf(" AND direction = $%d", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		q += fmt.Sprintf(" AND status = $%d", len(args))
	}
	args = append(args, f.Limit)
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))
	args = append(args, f.Offset)
	q += fmt.Sprintf(" OFFSET $%d", len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}
	defer rows.Close()
	out := []*invoice.Invoice{}
	for rows.Next() {
		var id uuid.UUID
		var payload []byte
		var status invoice.Status
		var created, updated time.Time
		if err := rows.Scan(&id, &payload, &status, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		inv := &invoice.Invoice{}
		if err := json.Unmarshal(payload, inv); err != nil {
			return nil, fmt.Errorf("decode: %w", err)
		}
		inv.ID = id.String()
		inv.OrganizationID = orgID.String()
		inv.Status = status
		inv.CreatedAt = created
		inv.UpdatedAt = updated
		out = append(out, inv)
	}
	return out, rows.Err()
}
