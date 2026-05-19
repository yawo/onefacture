// Package middleware bundles HTTP middlewares for the onefacture gateway.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/yawo/onefacture/internal/gateway/problem"
	"github.com/yawo/onefacture/internal/storage"
)

type ctxKey int

const (
	ctxKeyOrgID ctxKey = iota
	ctxKeyAPIKey
)

// APIKeyAuth authenticates requests via the X-API-Key header.
type APIKeyAuth struct {
	store  *storage.Store
	pepper string
}

func NewAPIKeyAuth(store *storage.Store) *APIKeyAuth {
	return &APIKeyAuth{store: store}
}

func (a *APIKeyAuth) WithPepper(pepper string) *APIKeyAuth {
	a.pepper = pepper
	return a
}

// Middleware authenticates the request and injects the organization id into the context.
func (a *APIKeyAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if key == "" {
			problem.Unauthorized(w, r, "missing X-API-Key header")
			return
		}
		row, err := a.store.APIKeys.Lookup(r.Context(), key, a.pepper)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				problem.Unauthorized(w, r, "invalid API key")
				return
			}
			problem.Internal(w, r, "auth lookup failed")
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyOrgID, row.OrganizationID)
		ctx = context.WithValue(ctx, ctxKeyAPIKey, row.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OrgID returns the organization id from the request context.
func OrgID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyOrgID).(uuid.UUID)
	return v, ok
}

// APIKeyID returns the API key id used to authenticate the request.
func APIKeyID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxKeyAPIKey).(uuid.UUID)
	return v, ok
}
