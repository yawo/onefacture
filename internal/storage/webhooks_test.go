package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWebhookHashSecret(t *testing.T) {
	secret := "my-webhook-secret"
	hash1 := HashSecret(secret)
	hash2 := HashSecret(secret)

	require.NotNil(t, hash1)
	require.Equal(t, hash1, hash2)
	require.Len(t, hash1, 32)
}

func TestWebhookHashSecretDifferent(t *testing.T) {
	hash1 := HashSecret("secret1")
	hash2 := HashSecret("secret2")

	require.NotEqual(t, hash1, hash2)
}

func TestWebhookCreateSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	ep, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret-key", []string{"invoice.submitted"})

	require.NoError(t, err)
	require.NotNil(t, ep)
	require.NotEqual(t, uuid.Nil, ep.ID)
	require.Equal(t, orgID, ep.OrganizationID)
	require.Equal(t, "https://example.com/webhook", ep.URL)
	require.True(t, ep.Active)
	require.Contains(t, ep.Events, "invoice.submitted")
	require.False(t, ep.CreatedAt.IsZero())
}

func TestWebhookCreateEmptyEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	ep, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret", []string{})

	require.NoError(t, err)
	require.NotNil(t, ep)
	require.Contains(t, ep.Events, "*")
}

func TestWebhookCreateMultipleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	events := []string{"invoice.submitted", "invoice.accepted", "invoice.rejected"}
	ep, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret", events)

	require.NoError(t, err)
	require.NotNil(t, ep)
	for _, e := range events {
		require.Contains(t, ep.Events, e)
	}
}

func TestWebhookGetEndpointSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	created, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret", []string{"invoice.submitted"})
	require.NoError(t, err)

	retrieved, err := store.Webhooks.GetEndpoint(ctx, created.ID)

	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.Equal(t, created.ID, retrieved.ID)
	require.Equal(t, created.OrganizationID, retrieved.OrganizationID)
	require.Equal(t, created.URL, retrieved.URL)
}

func TestWebhookGetEndpointNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	_, err := store.Webhooks.GetEndpoint(ctx, uuid.New())

	require.Equal(t, ErrNotFound, err)
}

func TestWebhookListActiveEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	endpoints, err := store.Webhooks.ListActive(ctx, uuid.New(), "invoice.submitted")

	require.NoError(t, err)
	require.Empty(t, endpoints)
}

func TestWebhookListActiveSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	ep, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret", []string{"invoice.submitted"})
	require.NoError(t, err)

	endpoints, err := store.Webhooks.ListActive(ctx, orgID, "invoice.submitted")

	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	require.Equal(t, ep.ID, endpoints[0].ID)
}

func TestWebhookListActiveWildcard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	ep, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret", []string{"*"})
	require.NoError(t, err)

	endpoints, err := store.Webhooks.ListActive(ctx, orgID, "invoice.submitted")

	require.NoError(t, err)
	require.Len(t, endpoints, 1)
	require.Equal(t, ep.ID, endpoints[0].ID)
}

func TestWebhookListActiveMultipleEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	ep, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook", "secret", 
		[]string{"invoice.submitted", "invoice.rejected"})
	require.NoError(t, err)

	endpoints1, err := store.Webhooks.ListActive(ctx, orgID, "invoice.submitted")
	require.NoError(t, err)
	require.Len(t, endpoints1, 1)
	require.Equal(t, ep.ID, endpoints1[0].ID)

	endpoints2, err := store.Webhooks.ListActive(ctx, orgID, "invoice.rejected")
	require.NoError(t, err)
	require.Len(t, endpoints2, 1)

	endpoints3, err := store.Webhooks.ListActive(ctx, orgID, "invoice.accepted")
	require.NoError(t, err)
	require.Empty(t, endpoints3)
}

func TestWebhookEnqueueSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	endpointID := uuid.New()
	deliveryID, err := store.Webhooks.Enqueue(ctx, endpointID, "invoice.submitted", map[string]any{
		"invoice_id": "inv-123",
		"status":     "submitted",
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, deliveryID)
}

func TestWebhookEnqueueEmptyPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	deliveryID, err := store.Webhooks.Enqueue(ctx, uuid.New(), "invoice.submitted", map[string]any{})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, deliveryID)
}

func TestWebhookEnqueueNilPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	deliveryID, err := store.Webhooks.Enqueue(ctx, uuid.New(), "invoice.submitted", nil)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, deliveryID)
}

func TestWebhookNextDueEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	deliveries, err := store.Webhooks.NextDue(ctx, 10)

	require.NoError(t, err)
	require.Empty(t, deliveries)
}

func TestWebhookMarkDeliveredSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	deliveryID, err := store.Webhooks.Enqueue(ctx, uuid.New(), "invoice.submitted", map[string]any{})
	require.NoError(t, err)

	err = store.Webhooks.MarkDelivered(ctx, deliveryID)

	require.NoError(t, err)
}

func TestWebhookMarkDeliveredNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	err := store.Webhooks.MarkDelivered(ctx, uuid.New())

	require.NoError(t, err)
}

func TestWebhookMarkFailedSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	deliveryID, err := store.Webhooks.Enqueue(ctx, uuid.New(), "invoice.submitted", map[string]any{})
	require.NoError(t, err)

	nextAttempt := time.Now().Add(5 * time.Minute)
	err = store.Webhooks.MarkFailed(ctx, deliveryID, 1, nextAttempt, "Connection timeout")

	require.NoError(t, err)
}

func TestWebhookMarkFailedMultipleAttempts(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	deliveryID, err := store.Webhooks.Enqueue(ctx, uuid.New(), "invoice.submitted", map[string]any{})
	require.NoError(t, err)

	for i := 1; i <= 5; i++ {
		nextAttempt := time.Now().Add(time.Duration(i*5) * time.Minute)
		err = store.Webhooks.MarkFailed(ctx, deliveryID, i, nextAttempt, "Retry")
		require.NoError(t, err)
	}
}

func TestWebhookCreateMultipleEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 5; i++ {
		_, err := store.Webhooks.Create(ctx, orgID, "https://example.com/webhook"+string(rune('0'+i)), "secret", []string{"invoice.submitted"})
		require.NoError(t, err)
	}

	endpoints, err := store.Webhooks.ListActive(ctx, orgID, "invoice.submitted")
	require.NoError(t, err)
	require.Len(t, endpoints, 5)
}

func TestWebhookIsolationByOrganization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	org1 := uuid.New()
	org2 := uuid.New()

	ep1, err := store.Webhooks.Create(ctx, org1, "https://example.com/webhook1", "secret", []string{"invoice.submitted"})
	require.NoError(t, err)
	ep2, err := store.Webhooks.Create(ctx, org2, "https://example.com/webhook2", "secret", []string{"invoice.submitted"})
	require.NoError(t, err)

	endpoints1, err := store.Webhooks.ListActive(ctx, org1, "invoice.submitted")
	require.NoError(t, err)
	require.Len(t, endpoints1, 1)
	require.Equal(t, ep1.ID, endpoints1[0].ID)

	endpoints2, err := store.Webhooks.ListActive(ctx, org2, "invoice.submitted")
	require.NoError(t, err)
	require.Len(t, endpoints2, 1)
	require.Equal(t, ep2.ID, endpoints2[0].ID)
}

func TestWebhookComplexPayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	payload := map[string]any{
		"invoice": map[string]any{
			"id":     "inv-123",
			"number": "INV-2024-001",
			"amount": 1500.50,
		},
		"organization": map[string]any{
			"id":    "org-456",
			"name":  "Acme Corp",
			"siren": "123456782",
		},
		"events": []any{
			map[string]any{"status": "submitted", "timestamp": "2024-01-01T00:00:00Z"},
			map[string]any{"status": "accepted", "timestamp": "2024-01-01T01:00:00Z"},
		},
	}

	deliveryID, err := store.Webhooks.Enqueue(ctx, uuid.New(), "invoice.submitted", payload)

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, deliveryID)
}

func TestWebhookEnqueueMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	endpointID := uuid.New()

	for i := 0; i < 10; i++ {
		deliveryID, err := store.Webhooks.Enqueue(ctx, endpointID, "invoice.submitted", map[string]any{
			"sequence": i,
		})
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, deliveryID)
	}
}

func TestWebhookCreateWithUnicodeURL(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t, ctx)
	defer cleanup()

	orgID := uuid.New()
	ep, err := store.Webhooks.Create(ctx, orgID, "https://例え.jp/webhook", "secret", []string{"invoice.submitted"})

	require.NoError(t, err)
	require.NotNil(t, ep)
	require.Equal(t, "https://例え.jp/webhook", ep.URL)
}
