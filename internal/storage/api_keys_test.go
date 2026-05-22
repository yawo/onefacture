package storage

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/yawo/onefacture/internal/config"
)

var sharedTestStore = struct {
	mu      sync.Mutex
	once    sync.Once
	store   *Store
	cleanup func()
	err     error
}{}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedTestStore.cleanup != nil {
		sharedTestStore.cleanup()
	}
	os.Exit(code)
}

func TestAPIKeyHashKey(t *testing.T) {
	plaintext := "ofx_abcd1234"
	pepper := "secret_pepper"

	hash1 := HashKey(plaintext, pepper)
	hash2 := HashKey(plaintext, pepper)

	require.NotNil(t, hash1)
	require.Equal(t, hash1, hash2, "hashing same input should produce same output")
	require.Len(t, hash1, 32, "SHA256 hash should be 32 bytes")
}

func TestAPIKeyHashKeyDifferentInputs(t *testing.T) {
	hash1 := HashKey("key1", "pepper")
	hash2 := HashKey("key2", "pepper")
	hash3 := HashKey("key1", "different_pepper")

	require.NotEqual(t, hash1, hash2, "different keys should produce different hashes")
	require.NotEqual(t, hash1, hash3, "different peppers should produce different hashes")
}

func TestAPIKeyGenerateSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID)

	plaintext, key, err := store.APIKeys.Generate(ctx, orgID, "test_key", pepper)

	require.NoError(t, err)
	require.NotNil(t, key)
	require.NotEmpty(t, plaintext)
	require.True(t, len(plaintext) > 4, "plaintext should be longer than prefix")
	require.Equal(t, orgID, key.OrganizationID)
	require.Equal(t, "test_key", key.Name)
	require.NotEqual(t, uuid.Nil, key.ID)
	require.False(t, key.CreatedAt.IsZero())

	lastFour := plaintext[len(plaintext)-4:]
	require.Equal(t, lastFour, key.LastFour)
	require.True(t, plaintext[:4] == "ofx_", "plaintext should start with ofx_ prefix")
}

func TestAPIKeyGenerateMultipleKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID)

	key1, _, _ := store.APIKeys.Generate(ctx, orgID, "key1", pepper)
	key2, _, _ := store.APIKeys.Generate(ctx, orgID, "key2", pepper)

	require.NotEqual(t, key1, key2, "generated keys should be unique")
}

func TestAPIKeyLookupSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID)

	plaintext, generated, err := store.APIKeys.Generate(ctx, orgID, "test_key", pepper)
	require.NoError(t, err)

	found, err := store.APIKeys.Lookup(ctx, plaintext, pepper)

	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, generated.ID, found.ID)
	require.Equal(t, generated.OrganizationID, found.OrganizationID)
	require.Equal(t, generated.Name, found.Name)
	require.Equal(t, generated.LastFour, found.LastFour)
	require.Nil(t, found.RevokedAt)
}

func TestAPIKeyLookupNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	_, err := store.APIKeys.Lookup(ctx, "ofx_nonexistent", "pepper")

	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyLookupWrongPepper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	createTestOrganization(ctx, t, store, orgID)

	plaintext, _, err := store.APIKeys.Generate(ctx, orgID, "test_key", "correct_pepper")
	require.NoError(t, err)

	_, err = store.APIKeys.Lookup(ctx, plaintext, "wrong_pepper")

	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyLookupRevoked(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID)

	plaintext, generated, err := store.APIKeys.Generate(ctx, orgID, "test_key", pepper)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID, generated.ID)
	require.NoError(t, err)

	_, err = store.APIKeys.Lookup(ctx, plaintext, pepper)
	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyRevokeSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID)

	_, generated, err := store.APIKeys.Generate(ctx, orgID, "test_key", pepper)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID, generated.ID)

	require.NoError(t, err)
}

func TestAPIKeyRevokeNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	err := store.APIKeys.Revoke(ctx, uuid.New(), uuid.New())

	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyRevokeTwice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID)

	_, generated, err := store.APIKeys.Generate(ctx, orgID, "test_key", pepper)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID, generated.ID)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID, generated.ID)
	require.Equal(t, ErrNotFound, err, "revoking already revoked key should return ErrNotFound")
}

func TestAPIKeyRevokeWrongOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID1 := uuid.New()
	orgID2 := uuid.New()
	pepper := "test_pepper"
	createTestOrganization(ctx, t, store, orgID1)

	_, generated, err := store.APIKeys.Generate(ctx, orgID1, "test_key", pepper)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID2, generated.ID)
	require.Equal(t, ErrNotFound, err, "revoking key from different organization should fail")
}

func TestAPIKeyGenerateEmptyName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	createTestOrganization(ctx, t, store, orgID)

	plaintext, key, err := store.APIKeys.Generate(ctx, orgID, "", "pepper")

	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "", key.Name)
	require.NotEmpty(t, plaintext)
}

func TestAPIKeyGenerateEmptyPepper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	createTestOrganization(ctx, t, store, orgID)

	plaintext, generated, err := store.APIKeys.Generate(ctx, orgID, "test_key", "")
	require.NoError(t, err)

	found, err := store.APIKeys.Lookup(ctx, plaintext, "")

	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, generated.ID, found.ID)
}

func createTestOrganization(ctx context.Context, t *testing.T, store *Store, orgID uuid.UUID) {
	t.Helper()
	_, err := store.Pool().Exec(
		ctx,
		`INSERT INTO organizations (id, name, siren, pa_id) VALUES ($1, $2, $3, $4)`,
		orgID,
		"Test Organization",
		"123456782",
		"chorus",
	)
	require.NoError(t, err)
}

func setupTestStore(t *testing.T) (*Store, func()) {
	sharedTestStore.mu.Lock()
	sharedTestStore.once.Do(func() {
		sharedTestStore.store, sharedTestStore.cleanup, sharedTestStore.err = newSharedTestStore()
	})
	if sharedTestStore.err != nil {
		sharedTestStore.mu.Unlock()
		require.NoError(t, sharedTestStore.err)
	}
	if err := resetTestStore(sharedTestStore.store); err != nil {
		sharedTestStore.mu.Unlock()
		require.NoError(t, err)
	}
	return sharedTestStore.store, func() {
		sharedTestStore.mu.Unlock()
	}
}

func newSharedTestStore() (*Store, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "onefacture_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, err
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, nil, err
	}

	dsn := "postgresql://postgres:password@" + host + ":" + port.Port() + "/onefacture_test?sslmode=disable"

	cfg := config.DatabaseConfig{
		DSN:            dsn,
		MaxConns:       10,
		ConnectTimeout: 5 * time.Second,
		StatementCache: true,
	}

	var store *Store
	deadline := time.Now().Add(30 * time.Second)
	for {
		store, err = New(ctx, cfg)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = container.Terminate(ctx)
			return nil, nil, err
		}
		select {
		case <-ctx.Done():
			_ = container.Terminate(context.Background())
			return nil, nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
	migration, err := os.ReadFile("migrations/0001_init.up.sql")
	if err != nil {
		store.Close()
		_ = container.Terminate(ctx)
		return nil, nil, err
	}
	_, err = store.Pool().Exec(ctx, string(migration))
	if err != nil {
		store.Close()
		_ = container.Terminate(ctx)
		return nil, nil, err
	}
	if err := relaxTestForeignKeys(ctx, store); err != nil {
		store.Close()
		_ = container.Terminate(ctx)
		return nil, nil, err
	}

	cleanup := func() {
		store.Close()
		termCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = container.Terminate(termCtx)
	}

	return store, cleanup, nil
}

func relaxTestForeignKeys(ctx context.Context, store *Store) error {
	_, err := store.Pool().Exec(ctx, `
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS api_keys_organization_id_fkey;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_organization_id_fkey;
ALTER TABLE idempotency_keys DROP CONSTRAINT IF EXISTS idempotency_keys_organization_id_fkey;
ALTER TABLE lifecycle_events DROP CONSTRAINT IF EXISTS lifecycle_events_invoice_id_fkey;
ALTER TABLE lifecycle_events DROP CONSTRAINT IF EXISTS lifecycle_events_organization_id_fkey;
ALTER TABLE audit_log DROP CONSTRAINT IF EXISTS audit_log_organization_id_fkey;
ALTER TABLE webhook_endpoints DROP CONSTRAINT IF EXISTS webhook_endpoints_organization_id_fkey;
ALTER TABLE webhook_deliveries DROP CONSTRAINT IF EXISTS webhook_deliveries_endpoint_id_fkey;
ALTER TABLE submission_dlq DROP CONSTRAINT IF EXISTS submission_dlq_organization_id_fkey;
ALTER TABLE submission_dlq DROP CONSTRAINT IF EXISTS submission_dlq_invoice_id_fkey;
`)
	return err
}

func resetTestStore(store *Store) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := store.Pool().Exec(ctx, `
DELETE FROM api_keys;
DELETE FROM idempotency_keys;
DELETE FROM lifecycle_events;
DELETE FROM audit_log;
DELETE FROM webhook_deliveries;
DELETE FROM webhook_endpoints;
DELETE FROM submission_dlq;
DELETE FROM invoices;
DELETE FROM organizations;
`)
	return err
}
