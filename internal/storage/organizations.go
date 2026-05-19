package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	SIREN     string
	PAID      string
	Settings  map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationRepo struct{ pool *pgxpool.Pool }

func (r *OrganizationRepo) Create(ctx context.Context, o *Organization) error {
	const q = `
INSERT INTO organizations (id, name, siren, pa_id, settings)
VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), COALESCE($5, '{}'::jsonb))
RETURNING created_at, updated_at`
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return r.pool.QueryRow(ctx, q, o.ID, o.Name, o.SIREN, o.PAID, o.Settings).
		Scan(&o.CreatedAt, &o.UpdatedAt)
}

func (r *OrganizationRepo) Get(ctx context.Context, id uuid.UUID) (*Organization, error) {
	const q = `SELECT id, name, COALESCE(siren,''), COALESCE(pa_id,''), settings, created_at, updated_at
FROM organizations WHERE id = $1`
	o := &Organization{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&o.ID, &o.Name, &o.SIREN, &o.PAID, &o.Settings, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return o, nil
}
