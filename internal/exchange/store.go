package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrNotFound = errors.New("not found")

// Store persists rates and job states.
type Store interface {
	SaveRate(ctx context.Context, rate *Rate) error
	GetRateForDay(ctx context.Context, base, quote string, day time.Time) (*Rate, error)
	SaveJob(ctx context.Context, job *Job) error
	GetJob(ctx context.Context, id string) (*Job, error)
}

type RedisStore struct {
	client    *redis.Client
	retention time.Duration
}

func NewRedisStore(client *redis.Client, retention time.Duration) *RedisStore {
	return &RedisStore{client: client, retention: retention}
}

func (s *RedisStore) SaveRate(ctx context.Context, rate *Rate) error {
	payload, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("marshal rate: %w", err)
	}
	key := s.rateKey(rate.Base, rate.Quote, rate.AsOf)
	if err := s.client.Set(ctx, key, payload, s.retention).Err(); err != nil {
		return fmt.Errorf("save rate: %w", err)
	}
	return nil
}

func (s *RedisStore) GetRateForDay(ctx context.Context, base, quote string, day time.Time) (*Rate, error) {
	key := s.rateKey(base, quote, day)
	data, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get rate: %w", err)
	}
	var rate Rate
	if err := json.Unmarshal(data, &rate); err != nil {
		return nil, fmt.Errorf("unmarshal rate: %w", err)
	}
	return &rate, nil
}

func (s *RedisStore) SaveJob(ctx context.Context, job *Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	if err := s.client.Set(ctx, s.jobKey(job.ID), payload, s.retention).Err(); err != nil {
		return fmt.Errorf("save job: %w", err)
	}
	return nil
}

func (s *RedisStore) GetJob(ctx context.Context, id string) (*Job, error) {
	data, err := s.client.Get(ctx, s.jobKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, fmt.Errorf("unmarshal job: %w", err)
	}
	return &job, nil
}

func (s *RedisStore) rateKey(base, quote string, day time.Time) string {
	day = day.UTC()
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)
	return fmt.Sprintf("exchange:rate:%s:%s:%s", base, quote, day.Format("2006-01-02"))
}

func (s *RedisStore) jobKey(id string) string {
	return fmt.Sprintf("exchange:job:%s", id)
}
