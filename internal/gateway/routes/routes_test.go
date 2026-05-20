package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yawo/onefacture/internal/adapters/registry"
)

func TestHealthHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	Health(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "ok")
}

func TestListPlatforms(t *testing.T) {
	deps := Dependencies{
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
		Registry: registry.NewDefault(slog.Default()),
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/platforms", nil)
	ListPlatforms(deps).ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "mock")
}


func TestListPlatformsEmpty(t *testing.T) {
reg := registry.NewDefault(slog.Default())
deps := Dependencies{
Logger:   slog.Default(),
Registry: reg,
}

rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodGet, "/v1/platforms", nil)
ListPlatforms(deps).ServeHTTP(rec, req)

require.Equal(t, http.StatusOK, rec.Code)
require.Contains(t, rec.Body.String(), "platforms")
}

func TestListPlatformsResponseStructure(t *testing.T) {
reg := registry.NewDefault(slog.Default())
deps := Dependencies{
Logger:   slog.Default(),
Registry: reg,
}

rec := httptest.NewRecorder()
req := httptest.NewRequest(http.MethodGet, "/v1/platforms", nil)
ListPlatforms(deps).ServeHTTP(rec, req)

require.Equal(t, http.StatusOK, rec.Code)

var result map[string]interface{}
err := json.NewDecoder(rec.Body).Decode(&result)
require.NoError(t, err)

platforms, ok := result["platforms"]
require.True(t, ok)
require.IsType(t, []interface{}{}, platforms)
}
