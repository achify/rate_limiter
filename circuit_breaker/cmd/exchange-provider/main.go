package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type response struct {
	Base      string             `json:"base"`
	Timestamp time.Time          `json:"timestamp"`
	Rates     map[string]float64 `json:"rates"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/rate", rateHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := getEnv("PROVIDER_ADDR", ":8081")
	log.Printf("exchange provider listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func rateHandler(w http.ResponseWriter, _ *http.Request) {
	if delayMs, err := strconv.Atoi(getEnv("PROVIDER_DELAY_MS", "0")); err == nil && delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	if getEnv("PROVIDER_FORCE_ERROR", "false") == "true" {
		http.Error(w, "upstream provider temporary failure", http.StatusServiceUnavailable)
		return
	}

	usdToRub, err := strconv.ParseFloat(getEnv("USD_RUB", "92.15"), 64)
	if err != nil {
		http.Error(w, "invalid USD_RUB env", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{
		Base:      "USD",
		Timestamp: time.Now().UTC(),
		Rates: map[string]float64{
			"RUB": usdToRub,
		},
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
