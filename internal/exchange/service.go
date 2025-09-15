package exchange

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type jobRequest struct {
	jobID       string
	base        string
	quote       string
	callbackURL string
}

type ServiceOption func(*Service)

type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type Service struct {
	provider       RateProvider
	store          Store
	notifier       Notifier
	processTimeout time.Duration
	clock          Clock

	queue chan jobRequest
	wg    sync.WaitGroup
	once  sync.Once
}

func WithProcessTimeout(d time.Duration) ServiceOption {
	return func(s *Service) { s.processTimeout = d }
}

func WithClock(clock Clock) ServiceOption {
	return func(s *Service) { s.clock = clock }
}

func WithQueueSize(size int) ServiceOption {
	return func(s *Service) { s.queue = make(chan jobRequest, size) }
}

func NewService(provider RateProvider, store Store, notifier Notifier, opts ...ServiceOption) *Service {
	s := &Service{
		provider:       provider,
		store:          store,
		notifier:       notifier,
		processTimeout: 2 * time.Minute,
		clock:          realClock{},
		queue:          make(chan jobRequest, 32),
	}
	for _, opt := range opts {
		opt(s)
	}
	s.wg.Add(1)
	go s.worker()
	return s
}

func (s *Service) Close() {
	s.once.Do(func() {
		close(s.queue)
		s.wg.Wait()
	})
}

func (s *Service) RequestRate(ctx context.Context, base, quote string) (*Job, error) {
	return s.requestRate(ctx, base, quote, "")
}

func (s *Service) RequestRateWithWebhook(ctx context.Context, base, quote, callbackURL string) (*Job, error) {
	if strings.TrimSpace(callbackURL) == "" {
		return nil, fmt.Errorf("callback url required")
	}
	return s.requestRate(ctx, base, quote, callbackURL)
}

func (s *Service) requestRate(ctx context.Context, base, quote, callbackURL string) (*Job, error) {
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)

	today := s.clock.Now().Truncate(24 * time.Hour)
	if rate, err := s.store.GetRateForDay(ctx, base, quote, today); err == nil {
		now := s.clock.Now()
		job := &Job{
			ID:        uuid.NewString(),
			Status:    JobStatusCompleted,
			Rate:      rate,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if callbackURL != "" {
			job.CallbackURL = callbackURL
		}
		if err := s.store.SaveJob(ctx, job); err != nil {
			return nil, err
		}
		if callbackURL != "" && s.notifier != nil {
			go s.dispatchWebhook(job)
		}
		return job, nil
	} else if err != nil && err != ErrNotFound {
		return nil, err
	}

	now := s.clock.Now()
	job := &Job{
		ID:          uuid.NewString(),
		Status:      JobStatusPending,
		CallbackURL: callbackURL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.SaveJob(ctx, job); err != nil {
		return nil, err
	}
	req := jobRequest{jobID: job.ID, base: base, quote: quote, callbackURL: callbackURL}
	select {
	case s.queue <- req:
		return job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Service) GetJob(ctx context.Context, id string) (*Job, error) {
	return s.store.GetJob(ctx, id)
}

func (s *Service) GetRateForDay(ctx context.Context, base, quote string, day time.Time) (*Rate, error) {
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)
	return s.store.GetRateForDay(ctx, base, quote, day.Truncate(24*time.Hour))
}

func (s *Service) worker() {
	defer s.wg.Done()
	for req := range s.queue {
		if err := s.handleJob(req); err != nil {
			log.Printf("process job %s failed: %v", req.jobID, err)
		}
	}
}

func (s *Service) handleJob(req jobRequest) error {
	ctx := context.Background()
	if s.processTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.processTimeout)
		defer cancel()
	}
	job, err := s.store.GetJob(ctx, req.jobID)
	if err != nil {
		return fmt.Errorf("load job: %w", err)
	}

	today := s.clock.Now().Truncate(24 * time.Hour)
	rate, err := s.store.GetRateForDay(ctx, req.base, req.quote, today)
	if err != nil && err != ErrNotFound {
		return fmt.Errorf("lookup cached rate: %w", err)
	}
	if err == ErrNotFound {
		rate, err = s.provider.FetchRate(ctx, req.base, req.quote)
		if err != nil {
			job.Status = JobStatusFailed
			job.Failure = err.Error()
			job.UpdatedAt = s.clock.Now()
			return s.store.SaveJob(ctx, job)
		}
		if rate.AsOf.IsZero() {
			rate.AsOf = today
		}
		if err := s.store.SaveRate(ctx, rate); err != nil {
			job.Status = JobStatusFailed
			job.Failure = fmt.Sprintf("save rate: %v", err)
			job.UpdatedAt = s.clock.Now()
			return s.store.SaveJob(ctx, job)
		}
	}
	job.Status = JobStatusCompleted
	job.Rate = rate
	job.Failure = ""
	job.UpdatedAt = s.clock.Now()
	if err := s.store.SaveJob(ctx, job); err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	if job.CallbackURL != "" && s.notifier != nil {
		if err := s.notifier.Notify(context.Background(), job); err != nil {
			job.Status = JobStatusFailed
			job.Failure = fmt.Sprintf("webhook: %v", err)
			job.UpdatedAt = s.clock.Now()
			if saveErr := s.store.SaveJob(context.Background(), job); saveErr != nil {
				log.Printf("save failed webhook job %s: %v", job.ID, saveErr)
			}
			return err
		}
	}
	return nil
}

func (s *Service) dispatchWebhook(job *Job) {
	if s.notifier == nil || job.CallbackURL == "" {
		return
	}
	if err := s.notifier.Notify(context.Background(), job); err != nil {
		log.Printf("webhook notify job %s failed: %v", job.ID, err)
	}
}
