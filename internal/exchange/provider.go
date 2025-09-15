package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RateProvider retrieves currency exchange rates from an upstream service.
type RateProvider interface {
	FetchRate(ctx context.Context, base, quote string) (*Rate, error)
}

// HTTPProvider implements RateProvider using a public HTTP API.
type HTTPProvider struct {
	client *http.Client
	host   string
}

// HTTPProviderOption configures HTTPProvider.
type HTTPProviderOption func(*HTTPProvider)

// WithHTTPProviderHost overrides the API host. Useful for tests.
func WithHTTPProviderHost(host string) HTTPProviderOption {
	return func(p *HTTPProvider) {
		p.host = host
	}
}

// NewHTTPProvider creates a provider that calls exchangerate.host by default.
func NewHTTPProvider(client *http.Client, opts ...HTTPProviderOption) *HTTPProvider {
	p := &HTTPProvider{
		client: client,
		host:   "https://api.exchangerate.host",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

type httpProviderResponse struct {
	Rates map[string]float64 `json:"rates"`
	Date  string             `json:"date"`
	Base  string             `json:"base"`
}

// FetchRate queries the upstream API for the provided pair.
func (p *HTTPProvider) FetchRate(ctx context.Context, base, quote string) (*Rate, error) {
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)

	endpoint := fmt.Sprintf("%s/latest", strings.TrimRight(p.host, "/"))
	q := url.Values{}
	q.Set("base", base)
	q.Set("symbols", quote)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	var payload httpProviderResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	value, ok := payload.Rates[quote]
	if !ok {
		return nil, fmt.Errorf("missing rate for %s", quote)
	}
	asOf, err := time.Parse("2006-01-02", payload.Date)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	return &Rate{Base: base, Quote: quote, Value: value, AsOf: asOf.UTC()}, nil
}
