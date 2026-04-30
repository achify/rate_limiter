package circuitbreaker

import (
	"testing"
	"time"
)

func TestOpensAfterFailureThreshold(t *testing.T) {
	cb := New(Config{FailureThreshold: 2, OpenTimeout: time.Second, HalfOpenMaxCalls: 1})

	if err := cb.Allow(); err != nil {
		t.Fatalf("allow failed in closed state: %v", err)
	}
	cb.OnFailure()

	if err := cb.Allow(); err != nil {
		t.Fatalf("allow failed before threshold: %v", err)
	}
	cb.OnFailure()

	if got := cb.State(); got != StateOpen {
		t.Fatalf("state = %s, want %s", got, StateOpen)
	}
	if err := cb.Allow(); err != ErrCircuitOpen {
		t.Fatalf("allow error = %v, want %v", err, ErrCircuitOpen)
	}
}

func TestHalfOpenThenCloseOnSuccess(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	cb := New(Config{FailureThreshold: 1, OpenTimeout: 5 * time.Second, HalfOpenMaxCalls: 2})
	cb.now = func() time.Time { return now }

	if err := cb.Allow(); err != nil {
		t.Fatalf("allow failed: %v", err)
	}
	cb.OnFailure()

	now = now.Add(6 * time.Second)
	if err := cb.Allow(); err != nil {
		t.Fatalf("allow in half-open failed: %v", err)
	}
	cb.OnSuccess()

	if got := cb.State(); got != StateHalfOpen {
		t.Fatalf("state after one success = %s, want %s", got, StateHalfOpen)
	}

	if err := cb.Allow(); err != nil {
		t.Fatalf("second half-open allow failed: %v", err)
	}
	cb.OnSuccess()

	if got := cb.State(); got != StateClosed {
		t.Fatalf("state after successful probes = %s, want %s", got, StateClosed)
	}
}

func TestHalfOpenFailureTripsBackToOpen(t *testing.T) {
	now := time.Date(2026, 4, 28, 12, 0, 0, 0, time.UTC)
	cb := New(Config{FailureThreshold: 1, OpenTimeout: time.Second, HalfOpenMaxCalls: 1})
	cb.now = func() time.Time { return now }

	if err := cb.Allow(); err != nil {
		t.Fatalf("allow failed: %v", err)
	}
	cb.OnFailure()

	now = now.Add(2 * time.Second)
	if err := cb.Allow(); err != nil {
		t.Fatalf("allow in half-open failed: %v", err)
	}
	cb.OnFailure()

	if got := cb.State(); got != StateOpen {
		t.Fatalf("state = %s, want %s", got, StateOpen)
	}
}
