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
	Active         bool
	CreatedAt      time.Time
	SecretHash     []byte
}

type WebhookDelivery struct {
	ID            uuid.UUID
	EndpointID    uuid.UUID
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

func HashSecret(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func (r *WebhookRepo) Create(ctx context.Context, orgID uuid.UUID, url, secret string, events []string) (*WebhookEndpoint, error) {
	if len(events) == 0 {
		events = []string{"*"}
	}
	ep := &WebhookEndpoint{
		ID: uuid.New(), OrganizationID: orgID, URL: url, Events: events, Active: true,
		SecretHash: HashSecret(secret),
	}
	const q = `INSERT INTO webhook_endpoints (id, organization_id, url, secret_hash, events)
VALUES ($1,$2,$3,$4,$5) RETURNING created_at`
	if err := r.pool.QueryRow(ctx, q, ep.ID, orgID, url, ep.SecretHash, events).Scan(&ep.CreatedAt); err != nil {
		return nil, fmt.Errorf("insert webhook: %w", err)
	}
	return ep, nil
}

func (r *WebhookRepo) ListActive(ctx context.Context, orgID uuid.UUID, eventType string) ([]*WebhookEndpoint, error) {
	const q = `SELECT id, organization_id, url, secret_hash, events, active, created_at
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
		if err := rows.Scan(&ep.ID, &ep.OrganizationID, &ep.URL, &ep.SecretHash, &ep.Events, &ep.Active, &ep.CreatedAt); err != nil {
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
SET status = CASE WHEN $3 < 8 THEN 'RETRYING' ELSE 'FAILED' END,
    attempts = $3, last_error = $4, next_attempt_at = $5
WHERE id = $1`
	_, err := r.pool.Exec(ctx, q, id, attempts, attempts, errMsg, nextAt)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}

// GetEndpoint returns an endpoint by id (including secret hash).
func (r *WebhookRepo) GetEndpoint(ctx context.Context, id uuid.UUID) (*WebhookEndpoint, error) {
	const q = `SELECT id, organization_id, url, secret_hash, events, active, created_at
FROM webhook_endpoints WHERE id = $1`
	ep := &WebhookEndpoint{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&ep.ID, &ep.OrganizationID, &ep.URL, &ep.SecretHash, &ep.Events, &ep.Active, &ep.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}
	return ep, nil
}
