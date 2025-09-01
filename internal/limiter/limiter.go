package limiter

import "context"

// RateLimiter defines interface for allowing operations
// based on a key, returning true if action is allowed.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}
