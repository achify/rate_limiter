package exchange_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/example/rate_limiter/internal/exchange"
)

type stubProvider struct {
	mu    sync.Mutex
	rate  *exchange.Rate
	calls int
	err   error
}

func (s *stubProvider) FetchRate(ctx context.Context, base, quote string) (*exchange.Rate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	rateCopy := *s.rate
	return &rateCopy, nil
}

type capturingNotifier struct {
	mu   sync.Mutex
	jobs []*exchange.Job
	err  error
}

func (c *capturingNotifier) Notify(ctx context.Context, job *exchange.Job) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	jobCopy := *job
	c.jobs = append(c.jobs, &jobCopy)
	return nil
}

func TestServiceProcessesJobAndCachesRate(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	provider := &stubProvider{rate: &exchange.Rate{Base: "EUR", Quote: "USD", Value: 1.1, AsOf: time.Now().Truncate(24 * time.Hour)}}
	notifier := &capturingNotifier{}
	store := exchange.NewRedisStore(redisClient, 24*time.Hour)

	service := exchange.NewService(provider, store, notifier, exchange.WithProcessTimeout(time.Second))
	defer service.Close()

	ctx := context.Background()
	job, err := service.RequestRate(ctx, "eur", "usd")
	if err != nil {
		t.Fatalf("request rate: %v", err)
	}
	if job.Status != exchange.JobStatusPending {
		t.Fatalf("expected pending job, got %s", job.Status)
	}

	waitForStatus(t, service, job.ID, exchange.JobStatusCompleted)

	cached, err := service.RequestRate(ctx, "eur", "usd")
	if err != nil {
		t.Fatalf("request cached rate: %v", err)
	}
	if cached.Status != exchange.JobStatusCompleted {
		t.Fatalf("expected completed status, got %s", cached.Status)
	}
	if cached.Rate == nil {
		t.Fatalf("expected cached rate in job")
	}

	provider.mu.Lock()
	calls := provider.calls
	provider.mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected provider called once, got %d", calls)
	}
}

func TestServiceDispatchesWebhook(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer redisClient.Close()

	rate := &exchange.Rate{Base: "EUR", Quote: "USD", Value: 1.15, AsOf: time.Now().Truncate(24 * time.Hour)}
	provider := &stubProvider{rate: rate}
	notifier := &capturingNotifier{}
	store := exchange.NewRedisStore(redisClient, 24*time.Hour)

	service := exchange.NewService(provider, store, notifier, exchange.WithProcessTimeout(time.Second))
	defer service.Close()

	ctx := context.Background()
	job, err := service.RequestRateWithWebhook(ctx, "eur", "usd", "https://example.com/callback")
	if err != nil {
		t.Fatalf("request rate with webhook: %v", err)
	}
	if job.Status != exchange.JobStatusPending {
		t.Fatalf("expected pending job, got %s", job.Status)
	}

	waitForStatus(t, service, job.ID, exchange.JobStatusCompleted)

	notifier.mu.Lock()
	defer notifier.mu.Unlock()
	if len(notifier.jobs) != 1 {
		t.Fatalf("expected webhook to be invoked once, got %d", len(notifier.jobs))
	}
	if notifier.jobs[0].Rate == nil || notifier.jobs[0].Rate.Value != rate.Value {
		t.Fatalf("webhook payload missing rate")
	}
}

func waitForStatus(t *testing.T, service *exchange.Service, jobID string, status exchange.JobStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := service.GetJob(context.Background(), jobID)
		if err == nil && job.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("job %s did not reach status %s", jobID, status)
}
