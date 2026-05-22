package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrIdempotencyConflict   = errors.New("idempotency key reused for a different request")
	ErrIdempotencyInProgress = errors.New("idempotency request is still in progress")
)

type IdempotencyRecord struct {
	ID           uuid.UUID
	Key          string
	Method       string
	Path         string
	RequestHash  string
	StatusCode   int
	ResponseBody []byte
	ResourceType string
	ResourceID   string
}

type IdempotencyRepo struct{ pool *pgxpool.Pool }

func (r *IdempotencyRepo) Reserve(ctx context.Context, orgID uuid.UUID, key, method, path, requestHash string) (*IdempotencyRecord, bool, error) {
	const insert = `INSERT INTO idempotency_keys (organization_id, key, method, path, request_hash)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (organization_id, key) DO NOTHING
RETURNING id, key, method, path, request_hash`
	rec := &IdempotencyRecord{}
	err := r.pool.QueryRow(ctx, insert, orgID, key, method, path, requestHash).Scan(
		&rec.ID, &rec.Key, &rec.Method, &rec.Path, &rec.RequestHash,
	)
	if err == nil {
		return rec, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, fmt.Errorf("reserve idempotency key: %w", err)
	}

	rec, err = r.Get(ctx, orgID, key)
	if err != nil {
		return nil, false, err
	}
	if rec.Method != method || rec.Path != path || rec.RequestHash != requestHash {
		return nil, false, ErrIdempotencyConflict
	}
	if rec.StatusCode == 0 || len(rec.ResponseBody) == 0 {
		return nil, false, ErrIdempotencyInProgress
	}
	return rec, false, nil
}

func (r *IdempotencyRepo) Store(ctx context.Context, orgID uuid.UUID, key string, statusCode int, responseBody []byte, resourceType, resourceID string) error {
	const q = `UPDATE idempotency_keys
SET status_code = $3, response_body = $4, resource_type = NULLIF($5,''), resource_id = NULLIF($6,''), updated_at = now()
WHERE organization_id = $1 AND key = $2`
	tag, err := r.pool.Exec(ctx, q, orgID, key, statusCode, responseBody, resourceType, resourceID)
	if err != nil {
		return fmt.Errorf("store idempotency response: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *IdempotencyRepo) Release(ctx context.Context, orgID uuid.UUID, key string) error {
	const q = `DELETE FROM idempotency_keys
WHERE organization_id = $1 AND key = $2 AND status_code IS NULL`
	if _, err := r.pool.Exec(ctx, q, orgID, key); err != nil {
		return fmt.Errorf("release idempotency key: %w", err)
	}
	return nil
}

func (r *IdempotencyRepo) Get(ctx context.Context, orgID uuid.UUID, key string) (*IdempotencyRecord, error) {
	const q = `SELECT id, key, method, path, request_hash, COALESCE(status_code,0),
COALESCE(response_body, '{}'::jsonb), COALESCE(resource_type,''), COALESCE(resource_id,'')
FROM idempotency_keys WHERE organization_id = $1 AND key = $2`
	rec := &IdempotencyRecord{}
	err := r.pool.QueryRow(ctx, q, orgID, key).Scan(
		&rec.ID, &rec.Key, &rec.Method, &rec.Path, &rec.RequestHash,
		&rec.StatusCode, &rec.ResponseBody, &rec.ResourceType, &rec.ResourceID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get idempotency key: %w", err)
	}
	return rec, nil
}
