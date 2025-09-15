package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/example/rate_limiter/internal/exchange"
)

type ExchangeHandler struct {
	service *exchange.Service
}

func NewExchangeHandler(service *exchange.Service) *ExchangeHandler {
	return &ExchangeHandler{service: service}
}

func (h *ExchangeHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{base}/{quote}/webhook", h.requestWithWebhook)
	r.Post("/{base}/{quote}/jobs", h.enqueue)
	r.Get("/jobs/{jobID}", h.jobStatus)
	r.Get("/{base}/{quote}/latest", h.latest)
	return r
}

func (h *ExchangeHandler) requestWithWebhook(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")
	var payload struct {
		CallbackURL string `json:"callback_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	job, err := h.service.RequestRateWithWebhook(r.Context(), base, quote, payload.CallbackURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.writeJobResponse(w, job)
}

func (h *ExchangeHandler) enqueue(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")
	job, err := h.service.RequestRate(r.Context(), base, quote)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.writeJobResponse(w, job)
}

func (h *ExchangeHandler) jobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		if errors.Is(err, exchange.ErrNotFound) {
			http.Error(w, "job not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (h *ExchangeHandler) latest(w http.ResponseWriter, r *http.Request) {
	base := chi.URLParam(r, "base")
	quote := chi.URLParam(r, "quote")
	rate, err := h.service.GetRateForDay(r.Context(), base, quote, time.Now())
	if err != nil {
		if errors.Is(err, exchange.ErrNotFound) {
			http.Error(w, "rate not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rate)
}

func (h *ExchangeHandler) writeJobResponse(w http.ResponseWriter, job *exchange.Job) {
	status := http.StatusOK
	if job.Status == exchange.JobStatusPending {
		status = http.StatusAccepted
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(job)
}
