package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter is a rate limiter backed by Redis.
type RedisRateLimiter struct {
	client *redis.Client
	limit  int
	ttl    time.Duration
}

// NewRedis creates new RedisRateLimiter.
func NewRedis(client *redis.Client, limit int, ttl time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{client: client, limit: limit, ttl: ttl}
}

// Allow returns true if key is within limit. It increments the counter and
// sets expiration if it's the first attempt.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	allowed, err := allowScript.Run(ctx, r.client, []string{key}, int(r.ttl.Seconds()), r.limit).Bool()
	if err != nil {
		return false, fmt.Errorf("redis eval: %w", err)
	}
	return allowed, nil
}

var allowScript = redis.NewScript(`
local current
current = redis.call('INCR', KEYS[1])
if tonumber(current) == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
if tonumber(current) > tonumber(ARGV[2]) then
    return 0
end
return 1
`)
