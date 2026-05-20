package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/yawo/onefacture/internal/config"
)

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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"

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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"

	key1, _, _ := store.APIKeys.Generate(ctx, orgID, "key1", pepper)
	key2, _, _ := store.APIKeys.Generate(ctx, orgID, "key2", pepper)

	require.NotEqual(t, key1, key2, "generated keys should be unique")
}

func TestAPIKeyLookupSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"

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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	_, err := store.APIKeys.Lookup(ctx, "ofx_nonexistent", "pepper")

	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyLookupWrongPepper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	plaintext, _, err := store.APIKeys.Generate(ctx, uuid.New(), "test_key", "correct_pepper")
	require.NoError(t, err)

	_, err = store.APIKeys.Lookup(ctx, plaintext, "wrong_pepper")

	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyLookupRevoked(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"

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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"

	_, generated, err := store.APIKeys.Generate(ctx, orgID, "test_key", pepper)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID, generated.ID)

	require.NoError(t, err)
}

func TestAPIKeyRevokeNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	err := store.APIKeys.Revoke(ctx, uuid.New(), uuid.New())

	require.Equal(t, ErrNotFound, err)
}

func TestAPIKeyRevokeTwice(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID := uuid.New()
	pepper := "test_pepper"

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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	orgID1 := uuid.New()
	orgID2 := uuid.New()
	pepper := "test_pepper"

	_, generated, err := store.APIKeys.Generate(ctx, orgID1, "test_key", pepper)
	require.NoError(t, err)

	err = store.APIKeys.Revoke(ctx, orgID2, generated.ID)
	require.Equal(t, ErrNotFound, err, "revoking key from different organization should fail")
}

func TestAPIKeyGenerateEmptyName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	plaintext, key, err := store.APIKeys.Generate(ctx, uuid.New(), "", "pepper")

	require.NoError(t, err)
	require.NotNil(t, key)
	require.Equal(t, "", key.Name)
	require.NotEmpty(t, plaintext)
}

func TestAPIKeyGenerateEmptyPepper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(ctx, t)
	defer cleanup()

	plaintext, generated, err := store.APIKeys.Generate(ctx, uuid.New(), "test_key", "")
	require.NoError(t, err)

	found, err := store.APIKeys.Lookup(ctx, plaintext, "")

	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, generated.ID, found.ID)
}

func setupTestStore(ctx context.Context, t *testing.T) (*Store, func()) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "onefacture_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := "postgresql://postgres:password@" + host + ":" + port.Port() + "/onefacture_test?sslmode=disable"

	cfg := config.DatabaseConfig{
		DSN:             dsn,
		MaxConns:        10,
		ConnectTimeout:  5 * time.Second,
		StatementCache:  true,
	}

	store, err := New(ctx, cfg)
	require.NoError(t, err)

	cleanup := func() {
		store.Close()
		container.Terminate(ctx)
	}

	return store, cleanup
}
