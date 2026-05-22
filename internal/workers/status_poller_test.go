package workers

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/core/invoice"
	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/storage"
)

// MockAdapter is a mock PAAdapter for testing.
type MockAdapter struct {
	mock.Mock
}

func (m *MockAdapter) Name() string {
	return "mock"
}

func (m *MockAdapter) Submit(ctx context.Context, inv *invoice.Invoice) (*adapters.SubmitResult, error) {
	args := m.Called(ctx, inv)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*adapters.SubmitResult), args.Error(1)
}

func (m *MockAdapter) GetStatus(ctx context.Context, paRef string) (*adapters.LifecycleEvent, error) {
	args := m.Called(ctx, paRef)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*adapters.LifecycleEvent), args.Error(1)
}

func (m *MockAdapter) Webhook(ctx context.Context, payload []byte) (*adapters.WebhookEvent, error) {
	args := m.Called(ctx, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*adapters.WebhookEvent), args.Error(1)
}

func (m *MockAdapter) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockInvoiceRepo is a mock for the invoice repository.
type MockInvoiceRepo struct {
	mock.Mock
}

func (m *MockInvoiceRepo) UpdateStatus(ctx context.Context, orgID, invoiceID uuid.UUID, status invoice.Status) error {
	args := m.Called(ctx, orgID, invoiceID, status)
	return args.Error(0)
}

// MockLifecycleRepo is a mock for the lifecycle repository.
type MockLifecycleRepo struct {
	mock.Mock
}

func (m *MockLifecycleRepo) Record(ctx context.Context, orgID, invoiceID uuid.UUID, ev storage.LifecycleEvent) error {
	args := m.Called(ctx, orgID, invoiceID, ev)
	return args.Error(0)
}

// MockEventBus is a mock event bus for testing.
type MockEventBus struct {
	mock.Mock
	rdb interface{}
}

func (m *MockEventBus) Publish(ctx context.Context, ev events.Event) error {
	args := m.Called(ctx, ev)
	return args.Error(0)
}

func (m *MockEventBus) Client() interface{} {
	return m.rdb
}

func (m *MockEventBus) Close() {
}

func TestNewStatusPoller(t *testing.T) {
	logger := slog.Default()
	store := &storage.Store{}
	reg := &registry.Registry{}
	bus := &events.Bus{}

	poller := NewStatusPoller(logger, store, reg, bus)

	require.NotNil(t, poller)
	require.Equal(t, logger, poller.logger)
	require.Equal(t, store, poller.store)
	require.Equal(t, reg, poller.registry)
	require.Equal(t, bus, poller.bus)
	require.Equal(t, 30*time.Second, poller.interval)
}

func TestStatusPollerRun(t *testing.T) {
	logger := slog.Default()
	mockStore := &storage.Store{}
	mockReg := &registry.Registry{}
	mockBus := &events.Bus{}

	poller := NewStatusPoller(logger, mockStore, mockReg, mockBus)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run should return when context is canceled
	poller.Run(ctx)

	// If we got here without hanging, the test passes
}

func TestStatusPollerInterval(t *testing.T) {
	logger := slog.Default()
	store := &storage.Store{}
	reg := &registry.Registry{}
	bus := &events.Bus{}

	poller := NewStatusPoller(logger, store, reg, bus)
	require.Equal(t, 30*time.Second, poller.interval)
}
