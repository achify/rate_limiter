package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRateHandlerSuccess(t *testing.T) {
	t.Setenv("PROVIDER_FORCE_ERROR", "false")
	t.Setenv("USD_RUB", "100.25")
	t.Setenv("PROVIDER_DELAY_MS", "0")

	req := httptest.NewRequest(http.MethodGet, "/rate", nil)
	w := httptest.NewRecorder()

	rateHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var payload response
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if payload.Rates["RUB"] != 100.25 {
		t.Fatalf("RUB rate = %v, want %v", payload.Rates["RUB"], 100.25)
	}
}

func TestRateHandlerForcedError(t *testing.T) {
	t.Setenv("PROVIDER_FORCE_ERROR", "true")
	defer os.Unsetenv("PROVIDER_FORCE_ERROR")

	req := httptest.NewRequest(http.MethodGet, "/rate", nil)
	w := httptest.NewRecorder()

	rateHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}
