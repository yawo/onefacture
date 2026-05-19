package registry

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters"
	"github.com/yawo/onefacture/internal/adapters/mock"
)

func TestNewDefault(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := NewDefault(logger)

	require.NotNil(t, reg)
	require.NotNil(t, reg.adapters)
	require.Greater(t, len(reg.adapters), 0)

	// Should have mock adapter
	names := reg.Names()
	require.Contains(t, names, "mock")
}

func TestRegisterAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := &Registry{
		adapters: map[string]adapters.PAAdapter{},
		logger:   logger,
	}

	mockAdapter := mock.New()
	reg.Register(mockAdapter)

	a, err := reg.Get("mock")
	require.NoError(t, err)
	require.NotNil(t, a)
	require.Equal(t, "mock", a.Name())
}

func TestGetAdapter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := NewDefault(logger)

	adapter, err := reg.Get("mock")
	require.NoError(t, err)
	require.NotNil(t, adapter)
	require.Equal(t, "mock", adapter.Name())
}

func TestGetAdapterNotFound(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := &Registry{
		adapters: map[string]adapters.PAAdapter{},
		logger:   logger,
	}

	adapter, err := reg.Get("nonexistent")
	require.Error(t, err)
	require.Nil(t, adapter)
	require.Contains(t, err.Error(), "unknown PA adapter")
}

func TestGetAdapterDefaultsMock(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := NewDefault(logger)

	// Empty name should default to mock
	adapter, err := reg.Get("")
	require.NoError(t, err)
	require.NotNil(t, adapter)
	require.Equal(t, "mock", adapter.Name())
}

func TestNames(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := NewDefault(logger)

	names := reg.Names()
	require.NotEmpty(t, names)
	require.Contains(t, names, "mock")
	require.Greater(t, len(names), 0)
}

func TestNamesEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := &Registry{
		adapters: map[string]adapters.PAAdapter{},
		logger:   logger,
	}

	names := reg.Names()
	require.Empty(t, names)
}

func TestRegisterOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := &Registry{
		adapters: map[string]adapters.PAAdapter{},
		logger:   logger,
	}

	adapter1 := mock.New()
	reg.Register(adapter1)

	// Re-register with a different adapter
	adapter2 := mock.New()
	reg.Register(adapter2)

	// Should return the latest registered
	adapter, err := reg.Get("mock")
	require.NoError(t, err)
	require.NotNil(t, adapter)
}

func TestRegistryConcurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	reg := NewDefault(logger)

	// This test checks that concurrent reads work
	done := make(chan bool, 2)

	go func() {
		_, _ = reg.Get("mock")
		done <- true
	}()

	go func() {
		_ = reg.Names()
		done <- true
	}()

	<-done
	<-done
}
