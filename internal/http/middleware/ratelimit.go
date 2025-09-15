package middleware

import (
	"net/http"

	"github.com/example/rate_limiter/internal/limiter"
)

// RateLimit returns an HTTP middleware that limits requests using the provided
// RateLimiter and keyFn. If l.Allow reports that the request is not allowed,
// the middleware responds with StatusTooManyRequests.
func RateLimit(l limiter.RateLimiter, keyFn func(*http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := keyFn(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			allowed, err := l.Allow(r.Context(), key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
