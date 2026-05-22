package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WebhookEndpoint struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	URL            string
	Events         []string
	IPAllowlist    []string
	MTLSRequired   bool
	MTLSCertRef    string
	Active         bool
	CreatedAt      time.Time
	SecretHash     []byte
}

type WebhookDelivery struct {
	ID            uuid.UUID
	EndpointID    uuid.UUID
	EndpointURL   string
	EventType     string
	Payload       map[string]any
	Status        string
	Attempts      int
	LastError     string
	NextAttemptAt *time.Time
	DeliveredAt   *time.Time
	CreatedAt     time.Time
}

type WebhookRepo struct{ pool *pgxpool.Pool }

type WebhookEndpointOptions struct {
	IPAllowlist  []string
	MTLSRequired bool
	MTLSCertRef  string
}

func HashSecret(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func (r *WebhookRepo) Create(ctx context.Context, orgID uuid.UUID, url, secret string, events []string) (*WebhookEndpoint, error) {
	return r.CreateWithOptions(ctx, orgID, url, secret, events, WebhookEndpointOptions{})
}

func (r *WebhookRepo) CreateWithOptions(ctx context.Context, orgID uuid.UUID, url, secret string, events []string, opts WebhookEndpointOptions) (*WebhookEndpoint, error) {
	if len(events) == 0 {
		events = []string{"*"}
	}
	if opts.IPAllowlist == nil {
		opts.IPAllowlist = []string{}
	}
	ep := &WebhookEndpoint{
		ID: uuid.New(), OrganizationID: orgID, URL: url, Events: events, Active: true,
		SecretHash: HashSecret(secret), IPAllowlist: opts.IPAllowlist, MTLSRequired: opts.MTLSRequired, MTLSCertRef: opts.MTLSCertRef,
	}
	const q = `INSERT INTO webhook_endpoints (id, organization_id, url, secret_hash, events, ip_allowlist, mtls_required, mtls_cert_ref)
VALUES ($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')) RETURNING created_at`
	if err := r.pool.QueryRow(ctx, q, ep.ID, orgID, url, ep.SecretHash, events, ep.IPAllowlist, ep.MTLSRequired, ep.MTLSCertRef).Scan(&ep.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert webhook: %w", err)
	}
	return ep, nil
}

func (r *WebhookRepo) ListActive(ctx context.Context, orgID uuid.UUID, eventType string) ([]*WebhookEndpoint, error) {
	const q = `SELECT id, organization_id, url, secret_hash, events, ip_allowlist, mtls_required, COALESCE(mtls_cert_ref,''), active, created_at
FROM webhook_endpoints WHERE organization_id = $1 AND active = TRUE
  AND ('*' = ANY(events) OR $2 = ANY(events))`
	rows, err := r.pool.Query(ctx, q, orgID, eventType)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()
	out := []*WebhookEndpoint{}
	for rows.Next() {
		ep := &WebhookEndpoint{}
		if err := rows.Scan(&ep.ID, &ep.OrganizationID, &ep.URL, &ep.SecretHash, &ep.Events, &ep.IPAllowlist, &ep.MTLSRequired, &ep.MTLSCertRef, &ep.Active, &ep.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		out = append(out, ep)
	}
	return out, rows.Err()
}

func (r *WebhookRepo) Enqueue(ctx context.Context, endpointID uuid.UUID, eventType string, payload map[string]any) (uuid.UUID, error) {
	id := uuid.New()
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal: %w", err)
	}
	const q = `INSERT INTO webhook_deliveries (id, endpoint_id, event_type, payload, status, next_attempt_at)
VALUES ($1,$2,$3,$4,'PENDING', now())`
	if _, err := r.pool.Exec(ctx, q, id, endpointID, eventType, payloadJSON); err != nil {
		return uuid.Nil, fmt.Errorf("enqueue delivery: %w", err)
	}
	return id, nil
}

func (r *WebhookRepo) NextDue(ctx context.Context, limit int) ([]WebhookDelivery, error) {
	const q = `SELECT id, endpoint_id, event_type, payload, status, attempts, COALESCE(last_error,''), next_attempt_at, delivered_at, created_at
FROM webhook_deliveries
WHERE status IN ('PENDING','RETRYING') AND next_attempt_at <= now()
ORDER BY next_attempt_at ASC LIMIT $1`
	rows, err := r.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("list due: %w", err)
	}
	defer rows.Close()
	out := []WebhookDelivery{}
	for rows.Next() {
		d := WebhookDelivery{}
		var payload []byte
		if err := rows.Scan(&d.ID, &d.EndpointID, &d.EventType, &payload, &d.Status, &d.Attempts, &d.LastError, &d.NextAttemptAt, &d.DeliveredAt, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		_ = json.Unmarshal(payload, &d.Payload)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *WebhookRepo) MarkDelivered(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE webhook_deliveries SET status='DELIVERED', delivered_at = now() WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark delivered: %w", err)
	}
	return nil
}

func (r *WebhookRepo) MarkFailed(ctx context.Context, id uuid.UUID, attempts int, nextAt time.Time, errMsg string) error {
	const q = `UPDATE webhook_deliveries
SET status = CASE WHEN $2 < 8 THEN 'RETRYING' ELSE 'FAILED' END,
    attempts = $2, last_error = $3, next_attempt_at = $4
WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id, attempts, errMsg, nextAt)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

func (r *WebhookRepo) ListDeliveries(ctx context.Context, orgID uuid.UUID, status string, limit int) ([]WebhookDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	const q = `SELECT d.id, d.endpoint_id, e.url, d.event_type, d.payload, d.status, d.attempts,
COALESCE(d.last_error,''), d.next_attempt_at, d.delivered_at, d.created_at
FROM webhook_deliveries d
JOIN webhook_endpoints e ON e.id = d.endpoint_id
WHERE e.organization_id = $1 AND (NULLIF($2,'') IS NULL OR d.status = $2)
ORDER BY d.created_at DESC LIMIT $3`
	rows, err := r.pool.Query(ctx, q, orgID, status, limit)
	if err != nil {
		return nil, fmt.Errorf("list webhook deliveries: %w", err)
	}
	defer rows.Close()
	out := []WebhookDelivery{}
	for rows.Next() {
		d := WebhookDelivery{}
		var payload []byte
		if err := rows.Scan(&d.ID, &d.EndpointID, &d.EndpointURL, &d.EventType, &payload, &d.Status, &d.Attempts, &d.LastError, &d.NextAttemptAt, &d.DeliveredAt, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook delivery: %w", err)
		}
		_ = json.Unmarshal(payload, &d.Payload)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *WebhookRepo) ReplayDelivery(ctx context.Context, orgID, id uuid.UUID) error {
	const q = `UPDATE webhook_deliveries d
SET status = 'PENDING', next_attempt_at = now(), last_error = NULL
FROM webhook_endpoints e
WHERE d.endpoint_id = e.id AND e.organization_id = $1 AND d.id = $2 AND d.status = 'FAILED'`
	tag, err := r.pool.Exec(ctx, q, orgID, id)
	if err != nil {
		return fmt.Errorf("replay webhook delivery: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetEndpoint returns an endpoint by id (including secret hash).
func (r *WebhookRepo) GetEndpoint(ctx context.Context, id uuid.UUID) (*WebhookEndpoint, error) {
	const q = `SELECT id, organization_id, url, secret_hash, events, ip_allowlist, mtls_required, COALESCE(mtls_cert_ref,''), active, created_at
FROM webhook_endpoints WHERE id = $1`
	ep := &WebhookEndpoint{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&ep.ID, &ep.OrganizationID, &ep.URL, &ep.SecretHash, &ep.Events, &ep.IPAllowlist, &ep.MTLSRequired, &ep.MTLSCertRef, &ep.Active, &ep.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}
	return ep, nil
}
