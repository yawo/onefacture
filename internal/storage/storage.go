// Package storage provides the postgres-backed persistence layer for onefacture.
package storage

import (
	"context"
	"errors"
	"fmt"

	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yawo/onefacture/internal/config"
)

// Store is the unified persistence facade.
type Store struct {
	pool *pgxpool.Pool

	Organizations *OrganizationRepo
	APIKeys       *APIKeyRepo
	Invoices      *InvoiceRepo
	Lifecycle     *LifecycleRepo
	Audit         *AuditRepo
	Webhooks      *WebhookRepo
}

// ErrNotFound is returned when a row is missing.
var ErrNotFound = errors.New("not found")

// New connects to PostgreSQL and returns a ready-to-use Store.
func New(ctx context.Context, cfg config.DatabaseConfig) (*Store, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout
	if !cfg.StatementCache {
		poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	s := &Store{pool: pool}
	s.Organizations = &OrganizationRepo{pool: pool}
	s.APIKeys = &APIKeyRepo{pool: pool}
	s.Invoices = &InvoiceRepo{pool: pool}
	s.Lifecycle = &LifecycleRepo{pool: pool}
	s.Audit = &AuditRepo{pool: pool}
	s.Webhooks = &WebhookRepo{pool: pool}
	return s, nil
}

// Pool returns the underlying pgx pool for advanced operations (migrations, tx).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// Close shuts down the connection pool.
func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
