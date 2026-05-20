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
	"github.com/yawo/onefacture/internal/core/invoice"
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
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
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

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dsn, cleanup, err := startPostgresContainer(ctx)
	require.NoError(t, err)
	defer cleanup()

	cfg := config.DatabaseConfig{
		DSN:             dsn,
		MaxConns:        25,
		ConnectTimeout:  5 * time.Second,
		StatementCache:  true,
	}

	store, err := New(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, store)
	defer store.Close()

	require.NotNil(t, store.Pool())
	require.NotNil(t, store.Organizations)
	require.NotNil(t, store.APIKeys)
	require.NotNil(t, store.Invoices)
	require.NotNil(t, store.Lifecycle)
	require.NotNil(t, store.Audit)
	require.NotNil(t, store.Webhooks)
}

func TestNewStoreInvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.DatabaseConfig{
		DSN:             "invalid-dsn",
		MaxConns:        25,
		ConnectTimeout:  1 * time.Second,
		StatementCache:  true,
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
		Active:         true,
		SecretHash:     HashSecret("secret-key"),
		CreatedAt:      time.Now(),
	}

	require.NotEqual(t, uuid.Nil, endpoint.ID)
	require.Equal(t, "https://example.com/webhook", endpoint.URL)
	require.True(t, endpoint.Active)
	require.Contains(t, endpoint.Events, "invoice.submitted")
	require.NotNil(t, endpoint.SecretHash)
}

func TestWebhookDeliveryStructure(t *testing.T) {
	now := time.Now()
	delivery := WebhookDelivery{
		ID:        uuid.New(),
		EndpointID: uuid.New(),
		EventType: "invoice.submitted",
		Payload:   map[string]any{"id": "inv-123"},
		Status:    "pending",
		Attempts:  0,
		CreatedAt: now,
	}

	require.NotEqual(t, uuid.Nil, delivery.ID)
	require.Equal(t, "invoice.submitted", delivery.EventType)
	require.Equal(t, "pending", delivery.Status)
	require.Equal(t, 0, delivery.Attempts)
}
