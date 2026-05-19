package openapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpecHandler(t *testing.T) {
	handler := SpecHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/yaml", w.Header().Get("Content-Type"))
	require.Greater(t, len(w.Body.String()), 0)
}

func TestSpecHandlerContent(t *testing.T) {
	handler := SpecHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)

	handler.ServeHTTP(w, r)

	body := w.Body.String()
	// Should contain OpenAPI version
	require.Contains(t, body, "openapi:")
	// Should contain some paths
	require.Contains(t, body, "/v1/")
}

func TestScalarHandler(t *testing.T) {
	handler := ScalarHandler("https://api.example.com")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/docs", nil)

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	require.Contains(t, w.Body.String(), "onefacture API")
}

func TestScalarHandlerIncludesURL(t *testing.T) {
	handler := ScalarHandler("https://custom.example.com/api")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/docs", nil)

	handler.ServeHTTP(w, r)

	body := w.Body.String()
	require.Contains(t, body, "https://custom.example.com/api/openapi.json")
	require.Contains(t, body, "@scalar/api-reference")
}

func TestScalarHandlerValidHTML(t *testing.T) {
	handler := ScalarHandler("http://localhost:8080")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/docs", nil)

	handler.ServeHTTP(w, r)

	body := w.Body.String()
	require.Contains(t, body, "<!doctype html>")
	require.Contains(t, body, "<html>")
	require.Contains(t, body, "</html>")
	require.Contains(t, body, "<script")
	require.Contains(t, body, "utf-8")
}

func TestScalarHandlerDifferentURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"localhost", "http://localhost:8080"},
		{"https", "https://api.onefacture.io"},
		{"custom port", "http://example.com:9000"},
		{"path", "http://example.com/api/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ScalarHandler(tt.url)
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/docs", nil)

			handler.ServeHTTP(w, r)

			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), tt.url)
		})
	}
}
