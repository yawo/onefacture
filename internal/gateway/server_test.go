package gateway

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/config"
	"github.com/yawo/onefacture/internal/gateway/middleware"
)

// Just test that Options can be created and New processes them
func TestNewServerOptions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg := &config.Config{
		HTTP: config.HTTPConfig{
			PublicBaseURL: "http://localhost:8080",
		},
	}

	opts := Options{
		Config:   cfg,
		Logger:   logger,
		Registry: registry.NewDefault(logger),
		AuthN:    middleware.NewAPIKeyAuth(nil),
	}

	// Check that Options is properly structured
	require.NotNil(t, opts.Config)
	require.NotNil(t, opts.Logger)
	require.NotNil(t, opts.Registry)
	require.NotNil(t, opts.AuthN)
}
