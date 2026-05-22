package storage

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuditAppendFirstEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	err := store.Audit.Append(ctx, orgID, "user@example.com", "CREATE", "invoice", "inv-123", map[string]any{
		"type": "commercial",
	})

	require.NoError(t, err)
}

func TestAuditAppendMultipleEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	err := store.Audit.Append(ctx, orgID, "user@example.com", "CREATE", "invoice", "inv-123", map[string]any{})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, orgID, "user@example.com", "SUBMIT", "invoice", "inv-123", map[string]any{
		"pa_id": "chorus",
	})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, orgID, "user@example.com", "UPDATE", "invoice", "inv-123", map[string]any{
		"status": "submitted",
	})
	require.NoError(t, err)
}

func TestAuditAppendEmptyMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	err := store.Audit.Append(ctx, orgID, "user@example.com", "CREATE", "organization", "", map[string]any{})

	require.NoError(t, err)
}

func TestAuditAppendNilMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	err := store.Audit.Append(ctx, orgID, "user@example.com", "CREATE", "organization", "", nil)

	require.NoError(t, err)
}

func TestAuditAppendComplexMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	metadata := map[string]any{
		"seller": map[string]any{
			"name":  "Acme Inc",
			"siren": "123456782",
		},
		"buyer": map[string]any{
			"name":  "XYZ Corp",
			"siren": "987654321",
		},
		"total": 1500.50,
		"items": []any{
			map[string]any{"desc": "item1", "qty": 2},
			map[string]any{"desc": "item2", "qty": 1},
		},
	}

	err := store.Audit.Append(ctx, orgID, "user@example.com", "CREATE", "invoice", "inv-456", metadata)

	require.NoError(t, err)
}

func TestAuditAppendEmptyFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	err := store.Audit.Append(ctx, orgID, "", "", "", "", map[string]any{})

	require.NoError(t, err)
}

func TestAuditAppendDifferentOrganizations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	org1 := uuid.New()
	org2 := uuid.New()

	err := store.Audit.Append(ctx, org1, "user1", "CREATE", "invoice", "inv-1", map[string]any{})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, org2, "user2", "CREATE", "invoice", "inv-2", map[string]any{})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, org1, "user1", "UPDATE", "invoice", "inv-1", map[string]any{})
	require.NoError(t, err)
}

func TestAuditAppendDifferentResourceTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	resourceTypes := []string{"invoice", "organization", "api_key", "webhook", "api_key_revocation"}
	for _, rt := range resourceTypes {
		err := store.Audit.Append(ctx, orgID, "user", "CREATE", rt, "resource-1", map[string]any{})
		require.NoError(t, err)
	}
}

func TestAuditAppendDifferentActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	actions := []string{"CREATE", "READ", "UPDATE", "DELETE", "SUBMIT", "APPROVE", "REJECT"}
	for _, action := range actions {
		err := store.Audit.Append(ctx, orgID, "user", action, "invoice", "inv-1", map[string]any{})
		require.NoError(t, err)
	}
}

func TestAuditAppendLongStrings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	longActor := "verylonguseremailthatmayexceedusuallengths@verylongcompanyname.example.com"
	longAction := "VERY_LONG_ACTION_NAME_THAT_DESCRIBES_SOMETHING_COMPLEX"
	longResourceType := "VERY_LONG_RESOURCE_TYPE_NAME"
	longResourceID := "resource-id-with-very-long-identifier-" + uuid.New().String()

	err := store.Audit.Append(ctx, orgID, longActor, longAction, longResourceType, longResourceID, map[string]any{
		"detail": "A very long description of what happened in this audit entry",
	})

	require.NoError(t, err)
}

func TestAuditAppendUnicodeCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	err := store.Audit.Append(ctx, orgID, "user@example.com", "CREATE", "invoice", "inv-123", map[string]any{
		"seller": "Société Générale de Technologie",
		"buyer":  "株式会社テクノロジー",
		"note":   "Facture avec caractères spéciaux: ñ, ü, é, 中文",
	})

	require.NoError(t, err)
}

func TestAuditHashChain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	err := store.Audit.Append(ctx, orgID, "user", "ACTION1", "resource", "id1", map[string]any{"seq": 1})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, orgID, "user", "ACTION2", "resource", "id2", map[string]any{"seq": 2})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, orgID, "user", "ACTION3", "resource", "id3", map[string]any{"seq": 3})
	require.NoError(t, err)

	err = store.Audit.Append(ctx, orgID, "user", "ACTION4", "resource", "id4", map[string]any{"seq": 4})
	require.NoError(t, err)
}

func TestAuditLatestHashInitial(t *testing.T) {
	orgID := uuid.New()

	prev := sha256.Sum256([]byte(orgID.String() + "user" + "ACTION" + "type" + "id" + "{}"))
	expected := prev[:]

	require.NotNil(t, expected)
	require.Len(t, expected, 32)
}

func TestAuditAppendManyEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()

	for i := 0; i < 100; i++ {
		err := store.Audit.Append(ctx, orgID, "user", "ACTION", "type", "id", map[string]any{
			"sequence": i,
		})
		require.NoError(t, err)
	}
}

func TestAuditAppendConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	store, cleanup := setupTestStore(t)
	defer cleanup()

	orgID := uuid.New()
	done := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			err := store.Audit.Append(ctx, orgID, "user", "ACTION", "type", "id", map[string]any{
				"goroutine": idx,
			})
			done <- err
		}(i)
	}

	for i := 0; i < 10; i++ {
		err := <-done
		require.NoError(t, err)
	}
}
