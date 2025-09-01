package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	httpserver "github.com/example/rate_limiter/internal/http"
	"github.com/example/rate_limiter/internal/limiter"
	"github.com/example/rate_limiter/internal/user"
)

func main() {
	ctx := context.Background()

	pgDSN := getenv("POSTGRES_DSN", "postgres://app:app@localhost:5432/app?sslmode=disable")
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	limit := getenvInt("RATE_LIMIT", 2)
	ttl := getenvDuration("RATE_LIMIT_TTL", 24*time.Hour)

	pool, err := pgxpool.New(ctx, pgDSN)
	if err != nil {
		log.Fatalf("pgxpool new: %v", err)
	}
	defer pool.Close()

	repo := user.NewPostgresRepository(pool)

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	rl := limiter.NewRedis(redisClient, limit, ttl)

	srv := httpserver.NewServer(repo, rl)

	addr := getenv("HTTP_ADDR", ":8080")
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
