package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/yawo/onefacture/internal/gateway/problem"
)

// RateLimit applies a per-organization, per-minute rate limit using a fixed-window
// counter in Redis. If the Redis client is nil, the middleware is a no-op.
type RateLimit struct {
	rdb     *redis.Client
	perMin  int
}

func NewRateLimit(rdb *redis.Client, perMin int) *RateLimit {
	return &RateLimit{rdb: rdb, perMin: perMin}
}

func (rl *RateLimit) Middleware(next http.Handler) http.Handler {
	if rl.rdb == nil || rl.perMin <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		org, ok := OrgID(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		minute := time.Now().UTC().Format("200601021504")
		key := fmt.Sprintf("rl:%s:%s", org.String(), minute)
		count, err := rl.rdb.Incr(r.Context(), key).Result()
		if err == nil && count == 1 {
			_ = rl.rdb.Expire(r.Context(), key, 90*time.Second).Err()
		}
		if err == nil && count > int64(rl.perMin) {
			w.Header().Set("Retry-After", "60")
			problem.TooMany(w, r, fmt.Sprintf("rate limit %d/min exceeded", rl.perMin))
			return
		}
		next.ServeHTTP(w, r)
	})
}
