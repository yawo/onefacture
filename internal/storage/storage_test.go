package storage

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/yawo/onefacture/internal/config"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/security"
)

// startPostgresContainer starts a PostgreSQL container for testing.
func startPostgresContainer(ctx context.Context) (string, func(), error) {
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
		return "", nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return "", nil, err
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		container.Terminate(ctx)
		return "", nil, err
	}

	dsn := "postgresql://postgres:password@" + host + ":" + port.Port() + "/onefacture_test?sslmode=disable"

	cleanup := func() {
		container.Terminate(ctx)
	}

	return dsn, cleanup, nil
}

func TestNewStoreSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	dsn, cleanup, err := startPostgresContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.DatabaseConfig{
		DSN:            dsn,
		MaxConns:       25,
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
			require.NoError(t, err)
		}
		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
	require.NotNil(t, store)
	defer store.Close()

	require.NotNil(t, store.Pool())
	require.NotNil(t, store.Organizations)
	require.NotNil(t, store.APIKeys)
	require.NotNil(t, store.Invoices)
	require.NotNil(t, store.Idempotency)
	require.NotNil(t, store.Lifecycle)
	require.NotNil(t, store.Submissions)
	require.NotNil(t, store.Audit)
	require.NotNil(t, store.Webhooks)
}

func TestNewStoreInvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.DatabaseConfig{
		DSN:            "invalid-dsn",
		MaxConns:       25,
		ConnectTimeout: 1 * time.Second,
		StatementCache: true,
	}

	_, err := New(ctx, cfg)
	require.Error(t, err)
}

func TestStoreCloseNil(t *testing.T) {
	store := &Store{pool: nil}
	store.Close() // should not panic
}

func TestErrNotFoundConstant(t *testing.T) {
	require.NotNil(t, ErrNotFound)
	require.Equal(t, "not found", ErrNotFound.Error())
}

func TestIdempotencyRecordStructure(t *testing.T) {
	rec := IdempotencyRecord{
		ID:           uuid.New(),
		Key:          "create-invoice-1",
		Method:       http.MethodPost,
		Path:         "/v1/invoices",
		RequestHash:  "abc123",
		StatusCode:   http.StatusCreated,
		ResponseBody: []byte(`{"id":"inv_123"}`),
		ResourceType: "invoice",
		ResourceID:   "inv_123",
	}

	require.NotEqual(t, uuid.Nil, rec.ID)
	require.Equal(t, http.StatusCreated, rec.StatusCode)
	require.Equal(t, "invoice", rec.ResourceType)
	require.NotEmpty(t, rec.ResponseBody)
}

func TestSubmissionDLQEntryStructure(t *testing.T) {
	entry := SubmissionDLQEntry{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		InvoiceID:      uuid.New(),
		PAID:           "chorus",
		Error:          "submit failed",
		Payload:        map[string]any{"attempt": 1},
		Status:         "FAILED",
		CreatedAt:      time.Now(),
	}

	require.NotEqual(t, uuid.Nil, entry.ID)
	require.Equal(t, "FAILED", entry.Status)
	require.Equal(t, "chorus", entry.PAID)
	require.Equal(t, 1, entry.Payload["attempt"])
}

func TestInvoiceRepoEncryptsAndDecryptsArtifacts(t *testing.T) {
	repo := &InvoiceRepo{encryptor: security.NewEncryptor(security.StaticKeyProvider{
		KeyID: "test-key",
		Key:   []byte("01234567890123456789012345678901"),
	})}
	orgID := uuid.New()
	invoiceID := uuid.New()
	raw := []byte("<xml>secret</xml>")

	encrypted, err := repo.encryptArtifact(context.Background(), orgID, invoiceID, "raw_xml", raw)
	require.NoError(t, err)
	require.NotEqual(t, raw, encrypted)
	require.Contains(t, string(encrypted), encryptedArtifactPrefix)

	metadata, err := InspectEncryptedArtifact("raw_xml", encrypted)
	require.NoError(t, err)
	require.True(t, metadata.Encrypted)
	require.Equal(t, "test-key", metadata.KeyID)
	require.Equal(t, "raw_xml", metadata.Field)

	decrypted, err := repo.decryptArtifact(context.Background(), orgID, invoiceID, "raw_xml", encrypted)
	require.NoError(t, err)
	require.Equal(t, raw, decrypted)
}

func TestInvoiceRepoLeavesArtifactsPlainWithoutEncryptor(t *testing.T) {
	repo := &InvoiceRepo{}
	raw := []byte("<xml>plain</xml>")

	got, err := repo.encryptArtifact(context.Background(), uuid.New(), uuid.New(), "raw_xml", raw)

	require.NoError(t, err)
	require.Equal(t, raw, got)
}

func TestOrganizationStructure(t *testing.T) {
	org := Organization{
		ID:        uuid.New(),
		Name:      "Test Org",
		SIREN:     "123456782",
		PAID:      "chorus",
		Settings:  map[string]any{"key": "value"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	require.NotEqual(t, uuid.Nil, org.ID)
	require.Equal(t, "Test Org", org.Name)
	require.Equal(t, "123456782", org.SIREN)
	require.Equal(t, "chorus", org.PAID)
	require.Equal(t, "value", org.Settings["key"])
}

func TestLifecycleEventStructure(t *testing.T) {
	ev := LifecycleEvent{
		FromStatus: invoice.StatusDraft,
		ToStatus:   invoice.StatusSubmitted,
		PACode:     "PA001",
		PAMessage:  "Invoice submitted successfully",
	}

	require.Equal(t, invoice.StatusDraft, ev.FromStatus)
	require.Equal(t, invoice.StatusSubmitted, ev.ToStatus)
	require.Equal(t, "PA001", ev.PACode)
	require.Equal(t, "Invoice submitted successfully", ev.PAMessage)
}

func TestWebhookEndpointStructure(t *testing.T) {
	endpoint := WebhookEndpoint{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		URL:            "https://example.com/webhook",
		Events:         []string{"invoice.submitted"},
		IPAllowlist:    []string{"127.0.0.1"},
		MTLSRequired:   true,
		MTLSCertRef:    "kms://webhook-client-cert",
		Active:         true,
		SecretHash:     HashSecret("secret-key"),
		CreatedAt:      time.Now(),
	}

	require.NotEqual(t, uuid.Nil, endpoint.ID)
	require.Equal(t, "https://example.com/webhook", endpoint.URL)
	require.True(t, endpoint.Active)
	require.Contains(t, endpoint.Events, "invoice.submitted")
	require.Contains(t, endpoint.IPAllowlist, "127.0.0.1")
	require.True(t, endpoint.MTLSRequired)
	require.NotNil(t, endpoint.SecretHash)
}

func TestWebhookDeliveryStructure(t *testing.T) {
	now := time.Now()
	delivery := WebhookDelivery{
		ID:         uuid.New(),
		EndpointID: uuid.New(),
		EventType:  "invoice.submitted",
		Payload:    map[string]any{"id": "inv-123"},
		Status:     "pending",
		Attempts:   0,
		CreatedAt:  now,
	}

	require.NotEqual(t, uuid.Nil, delivery.ID)
	require.Equal(t, "invoice.submitted", delivery.EventType)
	require.Equal(t, "pending", delivery.Status)
	require.Equal(t, 0, delivery.Attempts)
}
