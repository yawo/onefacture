package middleware

import (
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

const requestIDHeader = "X-Request-ID"

// RequestIDHeader mirrors chi's request ID context value into the HTTP response.
func RequestIDHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := chimw.GetReqID(r.Context()); id != "" {
			w.Header().Set(requestIDHeader, id)
		}
		next.ServeHTTP(w, r)
	})
}
