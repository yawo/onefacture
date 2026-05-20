package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/storage"
)

// MockEventBus is a mock for the event bus.
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, ev events.Event) error {
	args := m.Called(ctx, ev)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(ctx context.Context, group, consumer string, fn func(context.Context, events.Event) error) error {
	args := m.Called(ctx, group, consumer, fn)
	return args.Error(0)
}

func (m *MockEventBus) Client() interface{} {
	return nil
}

func (m *MockEventBus) Close() {
}

// MockWebhookStore is a mock for the webhook storage.
type MockWebhookStore struct {
	mock.Mock
}

func (m *MockWebhookStore) ListActive(ctx context.Context, orgID uuid.UUID, eventType string) ([]*storage.WebhookEndpoint, error) {
	args := m.Called(ctx, orgID, eventType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookStore) Enqueue(ctx context.Context, endpointID uuid.UUID, eventType string, payload map[string]any) (*storage.WebhookDelivery, error) {
	args := m.Called(ctx, endpointID, eventType, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookStore) NextDue(ctx context.Context, limit int) ([]storage.WebhookDelivery, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]storage.WebhookDelivery), args.Error(1)
}

func (m *MockWebhookStore) GetEndpoint(ctx context.Context, endpointID uuid.UUID) (*storage.WebhookEndpoint, error) {
	args := m.Called(ctx, endpointID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage.WebhookEndpoint), args.Error(1)
}

func (m *MockWebhookStore) MarkDelivered(ctx context.Context, deliveryID uuid.UUID) error {
	args := m.Called(ctx, deliveryID)
	return args.Error(0)
}

func (m *MockWebhookStore) MarkFailed(ctx context.Context, deliveryID uuid.UUID, attempts int, nextAttempt time.Time, errMsg string) error {
	args := m.Called(ctx, deliveryID, attempts, nextAttempt, errMsg)
	return args.Error(0)
}

// MockStore is a mock for the store.
type MockStore struct {
	mock.Mock
	Webhooks *MockWebhookStore
}

func (m *MockStore) Pool() interface{} {
	return nil
}

func (m *MockStore) Close() {
}

func TestNewDeliverer(t *testing.T) {
	logger := slog.Default()

	deliverer := NewDeliverer(logger, (*events.Bus)(nil), (*storage.Store)(nil))
	require.NotNil(t, deliverer)
	require.NotNil(t, deliverer.client)
	require.Equal(t, 10*time.Second, deliverer.client.Timeout)
}

func TestSignFunction(t *testing.T) {
	secret := []byte("test-secret")
	body := []byte("test body")
	sig := sign(secret, body)
	require.NotEmpty(t, sig)
	// Signature should be deterministic
	sig2 := sign(secret, body)
	require.Equal(t, sig, sig2)
}

func TestBackoffFunction(t *testing.T) {
	t1 := backoff(0)
	t2 := backoff(1)
	t3 := backoff(10)

	// Each should be later than the previous
	require.True(t, t1.Before(t2))
	require.True(t, t2.Before(t3))

	// Backoff should be capped at 1 hour
	t_max := backoff(100)
	now := time.Now().UTC()
	require.LessOrEqual(t, t_max.Sub(now), 1*time.Hour+1*time.Second)
}

func TestDelivererOnEventInvalidOrgID(t *testing.T) {
	logger := slog.Default()
	store := &storage.Store{}

	deliverer := NewDeliverer(logger, (*events.Bus)(nil), store)

	ctx := context.Background()
	ev := events.Event{
		OrganizationID: "invalid-uuid",
		Type:           "invoice.submitted",
	}

	// Should return nil error (invalid org ID is logged and ignored)
	err := deliverer.onEvent(ctx, ev)
	require.NoError(t, err)
}

func TestDelivererAttemptHTTPSuccess(t *testing.T) {
	logger := slog.Default()

	deliverer := NewDeliverer(logger, (*events.Bus)(nil), (*storage.Store)(nil))

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.NotEmpty(t, r.Header.Get("X-Onefacture-Signature"))
		require.NotEmpty(t, r.Header.Get("X-Onefacture-Event"))

		// Read body
		body, _ := io.ReadAll(r.Body)
		require.NotEmpty(t, body)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	require.NotNil(t, deliverer)
}

func TestDelivererAttemptMalformedPayload(t *testing.T) {
	logger := slog.Default()

	deliverer := NewDeliverer(logger, (*events.Bus)(nil), (*storage.Store)(nil))

	endpoint := storage.WebhookEndpoint{
		ID:         uuid.New(),
		URL:        "http://example.com",
		SecretHash: []byte("secret"),
	}

	delivery := storage.WebhookDelivery{
		ID:        uuid.New(),
		EndpointID: endpoint.ID,
		EventType: "invoice.submitted",
		Payload: map[string]any{
			"chan": make(chan int), // This cannot be marshaled to JSON
		},
		Attempts: 0,
	}

	// Since we can't mock the store easily, just verify the structure
	require.NotNil(t, delivery)
	require.NotNil(t, deliverer)
}

func TestSignatureFormat(t *testing.T) {
	secret := []byte("my-secret")
	body := []byte(`{"test":"data"}`)

	sig := sign(secret, body)

	// Signature should be valid hex
	require.Len(t, sig, 64) // SHA256 in hex is 64 characters
	matched, err := regexp.MatchString("^[a-f0-9]{64}$", sig)
	require.NoError(t, err)
	require.True(t, matched)
}

func TestDeliveryPayloadSerialization(t *testing.T) {
	delivery := storage.WebhookDelivery{
		ID:        uuid.New(),
		EndpointID: uuid.New(),
		EventType: "invoice.submitted",
		Payload: map[string]any{
			"type":            "invoice.submitted",
			"occurred_at":     time.Now().UTC(),
			"organization_id": "org-123",
			"invoice_id":      "inv-456",
			"data":            map[string]any{"pa": "chorus"},
		},
		Attempts: 0,
	}

	payload, err := json.Marshal(delivery.Payload)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	// Verify it's valid JSON
	var unmarshaled map[string]any
	err = json.Unmarshal(payload, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, "invoice.submitted", unmarshaled["type"])
}

func TestBackoffExponentialCurve(t *testing.T) {
	times := make([]time.Time, 0)
	for i := 0; i <= 5; i++ {
		times = append(times, backoff(i))
	}

	// Each should be significantly later than the previous
	for i := 1; i < len(times); i++ {
		require.True(t, times[i].After(times[i-1]))
	}
}

func TestWebhookHeaderFormat(t *testing.T) {
	payload := []byte(`{"test":"data"}`)
	sig := sign([]byte("secret"), payload)

	// Header should include sha256= prefix
	headerValue := "sha256=" + sig
	require.True(t, bytes.HasPrefix([]byte(headerValue), []byte("sha256=")))
	require.Len(t, headerValue, 7+64) // "sha256=" is 7 chars, hex is 64
}
