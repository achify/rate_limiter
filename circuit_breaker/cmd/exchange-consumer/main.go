package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/example/rate_limiter/circuit_breaker/internal/circuitbreaker"
)

type providerResponse struct {
	Base      string             `json:"base"`
	Timestamp time.Time          `json:"timestamp"`
	Rates     map[string]float64 `json:"rates"`
}

type consumerServer struct {
	providerURL string
	httpClient  *http.Client
	breaker     *circuitbreaker.CircuitBreaker
	fallback    float64
}

func main() {
	providerURL := getEnv("PROVIDER_URL", "http://localhost:8081/rate")
	timeoutMs := mustAtoiEnv("UPSTREAM_TIMEOUT_MS", 500)
	failureThreshold := uint(mustAtoiEnv("CB_FAILURE_THRESHOLD", 3))
	openTimeoutMs := mustAtoiEnv("CB_OPEN_TIMEOUT_MS", 3000)
	halfOpenCalls := uint(mustAtoiEnv("CB_HALF_OPEN_MAX_CALLS", 1))
	fallback := mustAtofEnv("FALLBACK_USD_RUB", 90.0)

	breaker := circuitbreaker.New(circuitbreaker.Config{
		FailureThreshold: failureThreshold,
		OpenTimeout:      time.Duration(openTimeoutMs) * time.Millisecond,
		HalfOpenMaxCalls: halfOpenCalls,
	})

	srv := &consumerServer{
		providerURL: providerURL,
		httpClient: &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond},
		breaker:    breaker,
		fallback:   fallback,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/converted", srv.convertedHandler)
	mux.HandleFunc("/breaker", srv.breakerHandler)

	addr := getEnv("CONSUMER_ADDR", ":8082")
	log.Printf("exchange consumer listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func (s *consumerServer) convertedHandler(w http.ResponseWriter, r *http.Request) {
	amount := 1.0
	if raw := r.URL.Query().Get("amount"); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || parsed <= 0 {
			http.Error(w, "amount must be positive number", http.StatusBadRequest)
			return
		}
		amount = parsed
	}

	rate, source, err := s.fetchRate(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	payload := map[string]any{
		"amount_usd": amount,
		"rate":       rate,
		"amount_rub": amount * rate,
		"source":     source,
		"state":      s.breaker.State(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *consumerServer) breakerHandler(w http.ResponseWriter, _ *http.Request) {
	payload := map[string]any{"state": s.breaker.State()}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *consumerServer) fetchRate(ctx context.Context) (rate float64, source string, err error) {
	if err := s.breaker.Allow(); err != nil {
		return s.fallback, "fallback_open_circuit", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.providerURL, nil)
	if err != nil {
		return 0, "", fmt.Errorf("build request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.breaker.OnFailure()
		return s.fallback, "fallback_upstream_error", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		s.breaker.OnFailure()
		return s.fallback, "fallback_upstream_error", nil
	}
	if resp.StatusCode != http.StatusOK {
		s.breaker.OnFailure()
		return 0, "", errors.New("unexpected upstream status")
	}

	var payload providerResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		s.breaker.OnFailure()
		return s.fallback, "fallback_upstream_error", nil
	}

	rate, ok := payload.Rates["RUB"]
	if !ok {
		s.breaker.OnFailure()
		return 0, "", errors.New("upstream response missing RUB")
	}

	s.breaker.OnSuccess()
	return rate, "upstream", nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func mustAtoiEnv(key string, fallback int) int {
	value := getEnv(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("invalid %s=%s", key, value)
	}
	return parsed
}

func mustAtofEnv(key string, fallback float64) float64 {
	value := getEnv(key, fmt.Sprintf("%.2f", fallback))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Fatalf("invalid %s=%s", key, value)
	}
	return parsed
}
