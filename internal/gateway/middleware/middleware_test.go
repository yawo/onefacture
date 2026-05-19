package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccessLogMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	middleware := AccessLog(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/test", nil)

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "Hello", w.Body.String())
}

func TestAccessLogRecordsStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	middleware := AccessLog(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not found"))
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/resource", nil)

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAccessLogDefaultsToOKStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	middleware := AccessLog(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't explicitly write header
		w.Write([]byte("OK"))
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(w, r)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestAccessLogRecordsSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	middleware := AccessLog(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("123456"))
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(w, r)

	require.Equal(t, 6, w.Body.Len())
}

func TestResponseWriterWrite(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder()}

	n, err := rw.Write([]byte("test"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, 4, rw.size)
	require.Equal(t, http.StatusOK, rw.status) // defaults to OK on first write
}

func TestResponseWriterWriteHeader(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: inner}

	rw.WriteHeader(http.StatusBadRequest)
	require.Equal(t, http.StatusBadRequest, rw.status)
	require.Equal(t, http.StatusBadRequest, inner.Code)
}

func TestResponseWriterMultipleWrites(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder()}

	n1, _ := rw.Write([]byte("hello"))
	n2, _ := rw.Write([]byte(" "))
	n3, _ := rw.Write([]byte("world"))

	require.Equal(t, 5, n1)
	require.Equal(t, 1, n2)
	require.Equal(t, 5, n3)
	require.Equal(t, 11, rw.size)
}

func TestOrgIDFromContext(t *testing.T) {
	testOrgID := uuid.New()
	ctx := context.WithValue(context.Background(), ctxKeyOrgID, testOrgID)

	orgID, ok := OrgID(ctx)
	require.True(t, ok)
	require.Equal(t, testOrgID, orgID)
}

func TestOrgIDNotInContext(t *testing.T) {
	ctx := context.Background()

	orgID, ok := OrgID(ctx)
	require.False(t, ok)
	require.Equal(t, uuid.UUID{}, orgID)
}

func TestOrgIDWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeyOrgID, "not-a-uuid")

	orgID, ok := OrgID(ctx)
	require.False(t, ok)
	require.Equal(t, uuid.UUID{}, orgID)
}

func TestAPIKeyIDFromContext(t *testing.T) {
	testKeyID := uuid.New()
	ctx := context.WithValue(context.Background(), ctxKeyAPIKey, testKeyID)

	keyID, ok := APIKeyID(ctx)
	require.True(t, ok)
	require.Equal(t, testKeyID, keyID)
}

func TestAPIKeyIDNotInContext(t *testing.T) {
	ctx := context.Background()

	keyID, ok := APIKeyID(ctx)
	require.False(t, ok)
	require.Equal(t, uuid.UUID{}, keyID)
}

func TestAPIKeyIDWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKeyAPIKey, 12345)

	keyID, ok := APIKeyID(ctx)
	require.False(t, ok)
	require.Equal(t, uuid.UUID{}, keyID)
}

func TestAccessLogRecordsTimer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	middleware := AccessLog(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(w, r)

	// Just verify it completes without error
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAccessLogVaryingMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	middleware := AccessLog(logger)

	for _, method := range methods {
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(method, "/test", nil)

		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)
	}
}

func TestAccessLogVaryingStatusCodes(t *testing.T) {
	codes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	for _, code := range codes {
		middleware := AccessLog(logger)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		handler.ServeHTTP(w, r)
		require.Equal(t, code, w.Code)
	}
}
