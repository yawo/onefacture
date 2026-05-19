package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditEntry is an immutable audit log record. A hash chain ties each entry
// to the prior one, so any tampering with the table is detectable.
type AuditEntry struct {
	ID             int64
	OrganizationID uuid.UUID
	Actor          string
	Action         string
	ResourceType   string
	ResourceID     string
	Metadata       map[string]any
	PrevHash       []byte
	RecordHash     []byte
	OccurredAt     time.Time
}

type AuditRepo struct{ pool *pgxpool.Pool }

// Append records an audit entry, computing the chain hash from the previous
// entry of the same organization.
func (r *AuditRepo) Append(ctx context.Context, orgID uuid.UUID, actor, action, resourceType, resourceID string, metadata map[string]any) error {
	prev, err := r.latestHash(ctx, orgID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("previous hash: %w", err)
	}
	now := time.Now().UTC()
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	hash := sha256.New()
	hash.Write(prev)
	hash.Write([]byte(orgID.String()))
	hash.Write([]byte(actor))
	hash.Write([]byte(action))
	hash.Write([]byte(resourceType))
	hash.Write([]byte(resourceID))
	hash.Write(metaJSON)
	hash.Write([]byte(now.Format(time.RFC3339Nano)))
	record := hash.Sum(nil)

	const q = `INSERT INTO audit_log
(organization_id, actor, action, resource_type, resource_id, metadata, prev_hash, record_hash, occurred_at)
VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9)`
	_, err = r.pool.Exec(ctx, q, orgID, actor, action, resourceType, resourceID, metaJSON, prev, record, now)
	if err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

func (r *AuditRepo) latestHash(ctx context.Context, orgID uuid.UUID) ([]byte, error) {
	const q = `SELECT record_hash FROM audit_log WHERE organization_id = $1 ORDER BY id DESC LIMIT 1`
	var h []byte
	err := r.pool.QueryRow(ctx, q, orgID).Scan(&h)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return h, nil
}
