package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/example/rate_limiter/internal/limiter"
	"github.com/example/rate_limiter/internal/user"
)

// UserHandler handles user related endpoints.
type UserHandler struct {
	repo    user.Repository
	limiter limiter.RateLimiter
}

func NewUserHandler(repo user.Repository, l limiter.RateLimiter) *UserHandler {
	return &UserHandler{repo: repo, limiter: l}
}

func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.list)
	r.Patch("/{userID}/change-password", h.changePassword)
	return r
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (h *UserHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "userID")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	allowed, err := h.limiter.Allow(r.Context(), limiterKey(id))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	if err := h.repo.ChangePassword(r.Context(), id, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func limiterKey(userID int) string {
	return "change_password:" + strconv.Itoa(userID)
}
