package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/rate_limiter/circuit_breaker/internal/circuitbreaker"
)

func TestConvertedHandlerUsesFallbackWhenOpen(t *testing.T) {
	var calls atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	srv := &consumerServer{
		providerURL: upstream.URL,
		httpClient:  &http.Client{Timeout: 100 * time.Millisecond},
		breaker: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 2,
			OpenTimeout:      10 * time.Second,
			HalfOpenMaxCalls: 1,
		}),
		fallback: 91.0,
	}

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/converted?amount=2", nil)
		w := httptest.NewRecorder()
		srv.convertedHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want %d", got, 2)
	}
}

func TestConvertedHandlerReturnsUpstreamWhenHealthy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"base":"USD","timestamp":"2026-04-28T12:00:00Z","rates":{"RUB":95.5}}`))
	}))
	defer upstream.Close()

	srv := &consumerServer{
		providerURL: upstream.URL,
		httpClient:  &http.Client{Timeout: 100 * time.Millisecond},
		breaker: circuitbreaker.New(circuitbreaker.Config{
			FailureThreshold: 2,
			OpenTimeout:      time.Second,
			HalfOpenMaxCalls: 1,
		}),
		fallback: 91.0,
	}

	req := httptest.NewRequest(http.MethodGet, "/converted?amount=3", nil)
	w := httptest.NewRecorder()
	srv.convertedHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload["source"] != "upstream" {
		t.Fatalf("source = %v, want upstream", payload["source"])
	}
}
