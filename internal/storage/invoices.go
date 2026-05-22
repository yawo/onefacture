package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/security"
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

const encryptedArtifactPrefix = "ofxenc1:"

type InvoiceRepo struct {
	pool      *pgxpool.Pool
	encryptor *security.Encryptor
}

type EncryptedArtifactMetadata struct {
	Encrypted bool   `json:"encrypted"`
	KeyID     string `json:"key_id,omitempty"`
	Field     string `json:"field,omitempty"`
}

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
	rawXML, err := r.encryptArtifact(ctx, orgID, id, "raw_xml", inv.RawXML)
	if err != nil {
		return uuid.Nil, err
	}
	rawPDF, err := r.encryptArtifact(ctx, orgID, id, "raw_pdf", inv.RawPDF)
	if err != nil {
		return uuid.Nil, err
	}
	err = r.pool.QueryRow(ctx, q,
		id, orgID, dir, inv.Status, inv.Profile, inv.TypeCode, inv.Number, inv.Currency,
		inv.IssueDate, due, inv.Seller.SIREN, inv.Buyer.SIREN, inv.PAID, inv.PARef,
		payload, rawXML, rawPDF,
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
	const q = `SELECT payload, status, raw_xml, raw_pdf, created_at, updated_at FROM invoices
WHERE id = $1 AND organization_id = $2`
	var payload []byte
	var status invoice.Status
	var rawXML, rawPDF []byte
	var created, updated time.Time
	err := r.pool.QueryRow(ctx, q, id, orgID).Scan(&payload, &status, &rawXML, &rawPDF, &created, &updated)
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
	inv.RawXML, err = r.decryptArtifact(ctx, orgID, id, "raw_xml", rawXML)
	if err != nil {
		return nil, err
	}
	inv.RawPDF, err = r.decryptArtifact(ctx, orgID, id, "raw_pdf", rawPDF)
	if err != nil {
		return nil, err
	}
	inv.ID = id.String()
	inv.OrganizationID = orgID.String()
	inv.CreatedAt = created
	inv.UpdatedAt = updated
	return inv, nil
}

func (r *InvoiceRepo) encryptArtifact(ctx context.Context, orgID, invoiceID uuid.UUID, field string, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 || r.encryptor == nil {
		return plaintext, nil
	}
	env, err := r.encryptor.Encrypt(ctx, plaintext, artifactAAD(orgID, invoiceID, field))
	if err != nil {
		return nil, fmt.Errorf("encrypt %s: %w", field, err)
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("marshal encrypted %s: %w", field, err)
	}
	return []byte(encryptedArtifactPrefix + string(raw)), nil
}

func (r *InvoiceRepo) decryptArtifact(ctx context.Context, orgID, invoiceID uuid.UUID, field string, raw []byte) ([]byte, error) {
	if len(raw) == 0 || !strings.HasPrefix(string(raw), encryptedArtifactPrefix) {
		return raw, nil
	}
	if r.encryptor == nil {
		return nil, fmt.Errorf("decrypt %s: encryption key not configured", field)
	}
	var env security.Envelope
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(raw), encryptedArtifactPrefix)), &env); err != nil {
		return nil, fmt.Errorf("decode encrypted %s: %w", field, err)
	}
	plain, err := r.encryptor.Decrypt(ctx, env, artifactAAD(orgID, invoiceID, field))
	if err != nil {
		return nil, fmt.Errorf("decrypt %s: %w", field, err)
	}
	return plain, nil
}

func InspectEncryptedArtifact(field string, raw []byte) (EncryptedArtifactMetadata, error) {
	if len(raw) == 0 || !strings.HasPrefix(string(raw), encryptedArtifactPrefix) {
		return EncryptedArtifactMetadata{Encrypted: false, Field: field}, nil
	}
	var env security.Envelope
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(raw), encryptedArtifactPrefix)), &env); err != nil {
		return EncryptedArtifactMetadata{}, fmt.Errorf("decode encrypted %s metadata: %w", field, err)
	}
	return EncryptedArtifactMetadata{Encrypted: true, KeyID: env.KeyID, Field: field}, nil
}

func artifactAAD(orgID, invoiceID uuid.UUID, field string) []byte {
	return []byte(orgID.String() + ":" + invoiceID.String() + ":" + field)
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

// SetSubmissionMetadata stores PA routing metadata on an invoice.
func (r *InvoiceRepo) SetSubmissionMetadata(ctx context.Context, orgID, id uuid.UUID, paID, paRef string) error {
	const q = `UPDATE invoices SET pa_id = NULLIF($1,''), pa_ref = NULLIF($2,''), updated_at = now() WHERE id = $3 AND organization_id = $4`
	tag, err := r.pool.Exec(ctx, q, paID, paRef, id, orgID)
	if err != nil {
		return fmt.Errorf("update submission metadata: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetLastRejection updates the invoice payload with the latest rejection details.
func (r *InvoiceRepo) SetLastRejection(ctx context.Context, orgID, id uuid.UUID, rej invoice.Rejection) error {
	inv, err := r.Get(ctx, orgID, id)
	if err != nil {
		return err
	}
	inv.LastRejection = &rej
	payload, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("marshal invoice: %w", err)
	}
	const q = `UPDATE invoices SET payload = $1, updated_at = now() WHERE id = $2 AND organization_id = $3`
	tag, err := r.pool.Exec(ctx, q, payload, id, orgID)
	if err != nil {
		return fmt.Errorf("update rejection payload: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InvoiceRepo) IncrementRejectionRetry(ctx context.Context, orgID, id uuid.UUID, resolutionHint string) error {
	inv, err := r.Get(ctx, orgID, id)
	if err != nil {
		return err
	}
	if inv.LastRejection == nil {
		return nil
	}
	now := time.Now().UTC()
	next := now.Add(30 * time.Minute)
	inv.LastRejection.RetryCount++
	inv.LastRejection.LastRetryAt = &now
	inv.LastRejection.NextRetryAt = &next
	if resolutionHint != "" {
		inv.LastRejection.ResolutionHint = resolutionHint
	}
	payload, err := json.Marshal(inv)
	if err != nil {
		return fmt.Errorf("marshal invoice: %w", err)
	}
	const q = `UPDATE invoices SET payload = $1, updated_at = now() WHERE id = $2 AND organization_id = $3`
	tag, err := r.pool.Exec(ctx, q, payload, id, orgID)
	if err != nil {
		return fmt.Errorf("update retry payload: %w", err)
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
