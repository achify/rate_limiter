package handler_test

import (
        "bytes"
        "context"
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
       rl := limiter.NewRedis(client, 2, time.Second)

       t.Cleanup(func() {
               client.FlushAll(context.Background())
               client.Close()
       })

       repo := user.NewMemoryRepository()
       return httpserver.NewServer(repo, rl)
}

func TestChangePasswordRateLimit(t *testing.T) {
	// Arrange
	srv := setupServer(t)

	makeReq := func() *httptest.ResponseRecorder {
		reqBody := bytes.NewBufferString(`{"password":"x"}`)
		req := httptest.NewRequest(http.MethodPatch, "/v1/users/1/change-password", reqBody)
		rr := httptest.NewRecorder()
		srv.ServeHTTP(rr, req)
		return rr
	}

	// Act
	first := makeReq()
	second := makeReq()
	third := makeReq()

	// Assert
	if first.Code != http.StatusOK {
		t.Fatalf("first request: expected %d, got %d", http.StatusOK, first.Code)
	}
	if second.Code != http.StatusOK {
		t.Fatalf("second request: expected %d, got %d", http.StatusOK, second.Code)
	}
	if third.Code != http.StatusTooManyRequests {
		t.Fatalf("third request: expected %d, got %d", http.StatusTooManyRequests, third.Code)
	}
}
