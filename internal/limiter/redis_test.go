package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisRateLimiter(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	limiter := NewRedis(client, 2, time.Hour)

	ctx := context.Background()

	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(ctx, "key")
		if err != nil {
			t.Fatalf("allow: %v", err)
		}
		if !allowed {
			t.Fatalf("expected allowed on iteration %d", i)
		}
	}
	allowed, err := limiter.Allow(ctx, "key")
	if err != nil {
		t.Fatalf("allow: %v", err)
	}
	if allowed {
		t.Fatalf("expected not allowed after limit")
	}
}
