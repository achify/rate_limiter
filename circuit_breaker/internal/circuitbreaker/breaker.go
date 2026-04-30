package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

type Config struct {
	FailureThreshold uint
	OpenTimeout      time.Duration
	HalfOpenMaxCalls uint
}

type CircuitBreaker struct {
	mu sync.Mutex

	cfg Config

	state      State
	failures   uint
	successes  uint
	halfCalls  uint
	openedAt   time.Time
	now        func() time.Time
}

func New(cfg Config) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 3
	}
	if cfg.OpenTimeout <= 0 {
		cfg.OpenTimeout = 5 * time.Second
	}
	if cfg.HalfOpenMaxCalls == 0 {
		cfg.HalfOpenMaxCalls = 1
	}

	return &CircuitBreaker{
		cfg:   cfg,
		state: StateClosed,
		now:   time.Now,
	}
}

func (cb *CircuitBreaker) Allow() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.now()

	switch cb.state {
	case StateOpen:
		if now.Sub(cb.openedAt) >= cb.cfg.OpenTimeout {
			cb.state = StateHalfOpen
			cb.halfCalls = 0
			cb.successes = 0
		} else {
			return ErrCircuitOpen
		}
	case StateHalfOpen:
		if cb.halfCalls >= cb.cfg.HalfOpenMaxCalls {
			return ErrCircuitOpen
		}
	}

	if cb.state == StateHalfOpen {
		cb.halfCalls++
	}

	return nil
}

func (cb *CircuitBreaker) OnSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		cb.failures = 0
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.cfg.HalfOpenMaxCalls {
			cb.state = StateClosed
			cb.resetCounters()
		}
	}
}

func (cb *CircuitBreaker) OnFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := cb.now()

	switch cb.state {
	case StateClosed:
		cb.failures++
		if cb.failures >= cb.cfg.FailureThreshold {
			cb.tripOpen(now)
		}
	case StateHalfOpen:
		cb.tripOpen(now)
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && cb.now().Sub(cb.openedAt) >= cb.cfg.OpenTimeout {
		cb.state = StateHalfOpen
		cb.halfCalls = 0
		cb.successes = 0
	}

	return cb.state
}

func (cb *CircuitBreaker) resetCounters() {
	cb.failures = 0
	cb.successes = 0
	cb.halfCalls = 0
}

func (cb *CircuitBreaker) tripOpen(now time.Time) {
	cb.state = StateOpen
	cb.openedAt = now
	cb.resetCounters()
}
