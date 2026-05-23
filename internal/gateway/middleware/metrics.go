package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/yawo/onefacture/internal/metrics"
)

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := &statusResponseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		path := r.URL.Path // could sanitize in real life

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(ww.status)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
