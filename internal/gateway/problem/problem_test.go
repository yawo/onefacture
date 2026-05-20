package problem

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	tests := []struct {
		name       string
		problem    Problem
		wantStatus int
		wantType   string
	}{
		{
			name: "default values",
			problem: Problem{
				Title: "Test Error",
			},
			wantStatus: http.StatusInternalServerError,
			wantType:   baseType + "internal",
		},
		{
			name: "custom status and type",
			problem: Problem{
				Type:   "custom-error",
				Status: http.StatusBadRequest,
				Title:  "Bad Request",
			},
			wantStatus: http.StatusBadRequest,
			wantType:   baseType + "custom-error",
		},
		{
			name: "http type not prefixed",
			problem: Problem{
				Type:   "https://example.com/error",
				Status: http.StatusNotFound,
				Title:  "Not Found",
			},
			wantStatus: http.StatusNotFound,
			wantType:   "https://example.com/error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/test", nil)

			Write(w, r, tt.problem)

			require.Equal(t, tt.wantStatus, w.Code)
			require.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
			require.Contains(t, w.Body.String(), tt.wantType)
			require.Contains(t, w.Body.String(), tt.problem.Title)
		})
	}
}

func TestWriteWithInstance(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/invoices/123", nil)

	Write(w, r, Problem{
		Type:   "test",
		Status: http.StatusOK,
	})

	require.Contains(t, w.Body.String(), "/v1/invoices/123")
}

func TestWriteWithoutRequest(t *testing.T) {
	w := httptest.NewRecorder()

	Write(w, nil, Problem{
		Type:   "test",
		Status: http.StatusOK,
	})

	require.Equal(t, http.StatusOK, w.Code)
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/test", nil)

	errs := []FieldError{
		{Field: "email", Code: "INVALID", Message: "Invalid email format"},
		{Field: "age", Code: "REQUIRED"},
	}

	BadRequest(w, r, "Validation failed", errs...)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "validation-failed")
	require.Contains(t, w.Body.String(), "email")
	require.Contains(t, w.Body.String(), "age")
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/test", nil)

	Unauthorized(w, r, "Missing API key")

	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "unauthorized")
	require.Contains(t, w.Body.String(), "Missing API key")
}

func TestForbidden(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/test", nil)

	Forbidden(w, r, "Access denied")

	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "forbidden")
	require.Contains(t, w.Body.String(), "Access denied")
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/invoices/999", nil)

	NotFound(w, r, "Invoice not found")

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "not-found")
}

func TestConflict(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/test", nil)

	Conflict(w, r, "Resource already exists")

	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "conflict")
}

func TestInternal(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1/test", nil)

	Internal(w, r, "Database connection failed")

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "internal")
}

func TestTooMany(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/test", nil)

	TooMany(w, r, "Rate limit exceeded")

	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.Contains(t, w.Body.String(), "rate-limited")
}

func TestNotImplemented(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/v1/test", nil)

	NotImplemented(w, r, "DELETE method not yet implemented")

	require.Equal(t, http.StatusNotImplemented, w.Code)
	require.Contains(t, w.Body.String(), "not-implemented")
}

func TestWriteWithErrors(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/v1/invoices", nil)

	errs := []FieldError{
		{Field: "seller.siren", Code: "INVALID", Message: "SIREN must be 9 digits"},
	}

	Write(w, r, Problem{
		Type:     "validation-failed",
		Status:   http.StatusBadRequest,
		Title:    "Validation Failed",
		Detail:   "Invalid invoice data",
		Errors:   errs,
		Instance: "/custom/path", // This should be overwritten with r.URL.Path
	})

	body := w.Body.String()
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, body, "seller.siren")
	require.Contains(t, body, "INVALID")
	require.Contains(t, body, "/v1/invoices") // Instance should be from request
}

func TestDefaultTitle(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	Write(w, r, Problem{
		Type:   "test",
		Status: http.StatusBadRequest,
	})

	// Should have default title for 400 status
	require.Contains(t, w.Body.String(), "Bad Request")
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	Write(w, r, Problem{
		Type:   "test",
		Title:  "Test",
		Status: http.StatusOK,
		Detail: "Test detail",
	})

	// Verify it's valid JSON
	require.Contains(t, w.Body.String(), "\"type\"")
	require.Contains(t, w.Body.String(), "\"title\"")
	require.Contains(t, w.Body.String(), "\"status\"")
	require.True(t, strings.HasPrefix(w.Body.String(), "{"), "Response should be JSON object")
}
