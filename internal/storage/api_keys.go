package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKey struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	LastFour       string
	RevokedAt      *time.Time
	CreatedAt      time.Time
}

type APIKeyRepo struct{ pool *pgxpool.Pool }

// HashKey returns the storage hash for a plaintext API key, peppered.
func HashKey(plaintext, pepper string) []byte {
	h := sha256.Sum256([]byte(plaintext + ":" + pepper))
	return h[:]
}

// Generate creates a new API key, returning the plaintext (only here) plus stored row.
func (r *APIKeyRepo) Generate(ctx context.Context, orgID uuid.UUID, name, pepper string) (plaintext string, key *APIKey, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("rand: %w", err)
	}
	plaintext = "ofx_" + hex.EncodeToString(raw)
	hash := HashKey(plaintext, pepper)
	lastFour := plaintext[len(plaintext)-4:]

	key = &APIKey{ID: uuid.New(), OrganizationID: orgID, Name: name, LastFour: lastFour}
	const q = `INSERT INTO api_keys (id, organization_id, name, key_hash, last_four)
VALUES ($1, $2, $3, $4, $5) RETURNING created_at`
	if err = r.pool.QueryRow(ctx, q, key.ID, key.OrganizationID, key.Name, hash, key.LastFour).Scan(&key.CreatedAt); err != nil {
		return "", nil, fmt.Errorf("insert api key: %w", err)
	}
	return plaintext, key, nil
}

// Lookup returns the matching API key row (if not revoked) given a plaintext key.
func (r *APIKeyRepo) Lookup(ctx context.Context, plaintext, pepper string) (*APIKey, error) {
	hash := HashKey(plaintext, pepper)
	const q = `SELECT id, organization_id, name, last_four, revoked_at, created_at
FROM api_keys WHERE key_hash = $1`
	k := &APIKey{}
	err := r.pool.QueryRow(ctx, q, hash).Scan(&k.ID, &k.OrganizationID, &k.Name, &k.LastFour, &k.RevokedAt, &k.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup api key: %w", err)
	}
	if k.RevokedAt != nil {
		return nil, ErrNotFound
	}
	return k, nil
}

// Revoke marks an API key as revoked.
func (r *APIKeyRepo) Revoke(ctx context.Context, orgID, keyID uuid.UUID) error {
	const q = `UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND organization_id = $2 AND revoked_at IS NULL`
	tag, err := r.pool.Exec(ctx, q, keyID, orgID)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
