package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/example/rate_limiter/internal/http/handler"
	"github.com/example/rate_limiter/internal/limiter"
	"github.com/example/rate_limiter/internal/user"
)

// NewServer builds HTTP server mux.
func NewServer(repo user.Repository, rl limiter.RateLimiter) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	userHandler := handler.NewUserHandler(repo, rl)
	r.Mount("/v1/users", userHandler.Routes())

	return r
}
