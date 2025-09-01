package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	httpserver "github.com/example/rate_limiter/internal/http"
	"github.com/example/rate_limiter/internal/limiter"
	"github.com/example/rate_limiter/internal/user"
)

func setupServer(t *testing.T) http.Handler {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rl := limiter.NewRedis(client, 2, 24*time.Hour)

	repo := user.NewMemoryRepository()
	return httpserver.NewServer(repo, rl)
}

func TestChangePasswordRateLimit(t *testing.T) {
	srv := setupServer(t)

	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/v1/users/1/change-password", bytes.NewBufferString(`{"password":"x"}`))
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		return rr
	}

	for i := 0; i < 2; i++ {
		if rr := makeReq(); rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	}
	if rr := makeReq(); rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
}
