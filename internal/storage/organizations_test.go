package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOrganizationCreateSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org := &Organization{
		Name:  "Test Organization",
		SIREN: "123456782",
		PAID:  "chorus",
		Settings: map[string]any{
			"auto_submit": true,
			"timeout_ms":  5000,
		},
	}

	err := store.Organizations.Create(ctx, org)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, org.ID)
	require.False(t, org.CreatedAt.IsZero())
	require.False(t, org.UpdatedAt.IsZero())
	require.Equal(t, org.CreatedAt, org.UpdatedAt)
}

func TestOrganizationCreateWithID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	id := uuid.New()
	org := &Organization{
		ID:    id,
		Name:  "Test Organization",
		SIREN: "123456782",
		PAID:  "chorus",
	}

	err := store.Organizations.Create(ctx, org)

	require.NoError(t, err)
	require.Equal(t, id, org.ID)
}

func TestOrganizationCreateEmptySIREN(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org := &Organization{
		Name:     "Test Organization",
		SIREN:    "",
		PAID:     "chorus",
		Settings: map[string]any{},
	}

	err := store.Organizations.Create(ctx, org)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, org.ID)
}

func TestOrganizationCreateEmptyPAID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org := &Organization{
		Name:     "Test Organization",
		SIREN:    "123456782",
		PAID:     "",
		Settings: map[string]any{},
	}

	err := store.Organizations.Create(ctx, org)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, org.ID)
}

func TestOrganizationCreateNilSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org := &Organization{
		Name:     "Test Organization",
		SIREN:    "123456782",
		PAID:     "chorus",
		Settings: nil,
	}

	err := store.Organizations.Create(ctx, org)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, org.ID)
}

func TestOrganizationGetSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	created := &Organization{
		Name:  "Test Organization",
		SIREN: "123456782",
		PAID:  "chorus",
		Settings: map[string]any{
			"key1": "value1",
		},
	}
	err := store.Organizations.Create(ctx, created)
	require.NoError(t, err)

	retrieved, err := store.Organizations.Get(ctx, created.ID)

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, created.ID, retrieved.ID)
	require.Equal(t, created.Name, retrieved.Name)
	require.Equal(t, created.SIREN, retrieved.SIREN)
	require.Equal(t, created.PAID, retrieved.PAID)
	require.Equal(t, created.Settings["key1"], retrieved.Settings["key1"])
	require.Equal(t, created.CreatedAt, retrieved.CreatedAt)
}

func TestOrganizationGetNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	_, err := store.Organizations.Get(ctx, uuid.New())

	require.Equal(t, ErrNotFound, err)
}

func TestOrganizationGetMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org1 := &Organization{
		Name:  "Organization 1",
		SIREN: "111111111",
		PAID:  "chorus",
	}
	org2 := &Organization{
		Name:  "Organization 2",
		SIREN: "222222222",
		PAID:  "docaposte",
	}

	err := store.Organizations.Create(ctx, org1)
	require.NoError(t, err)
	err = store.Organizations.Create(ctx, org2)
	require.NoError(t, err)

	retrieved1, err := store.Organizations.Get(ctx, org1.ID)
	require.NoError(t, err)
	require.Equal(t, org1.Name, retrieved1.Name)

	retrieved2, err := store.Organizations.Get(ctx, org2.ID)
	require.NoError(t, err)
	require.Equal(t, org2.Name, retrieved2.Name)
}

func TestOrganizationCreateEmptyName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org := &Organization{
		Name:     "",
		SIREN:    "123456782",
		PAID:     "chorus",
		Settings: map[string]any{},
	}

	err := store.Organizations.Create(ctx, org)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, org.ID)
}

func TestOrganizationGetEmptyFieldsCoalesced(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	created := &Organization{
		Name:  "Test Organization",
		SIREN: "",
		PAID:  "",
	}
	err := store.Organizations.Create(ctx, created)
	require.NoError(t, err)

	retrieved, err := store.Organizations.Get(ctx, created.ID)

	require.NoError(t, err)
	require.Equal(t, "", retrieved.SIREN)
	require.Equal(t, "", retrieved.PAID)
}

func TestOrganizationCreateComplexSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	complexSettings := map[string]any{
		"feature_flags": map[string]any{
			"beta": true,
			"experimental": false,
		},
		"timeout_ms": 5000,
		"auto_submit": true,
		"webhook_urls": []string{"https://example.com/webhook1", "https://example.com/webhook2"},
	}

	org := &Organization{
		Name:     "Test Organization",
		SIREN:    "123456782",
		PAID:     "chorus",
		Settings: complexSettings,
	}

	err := store.Organizations.Create(ctx, org)
	require.NoError(t, err)

	retrieved, err := store.Organizations.Get(ctx, org.ID)
	require.NoError(t, err)

	require.NotNil(t, retrieved.Settings)
	require.Equal(t, 5000, int(retrieved.Settings["timeout_ms"].(float64)))
	require.True(t, retrieved.Settings["auto_submit"].(bool))
}
