package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Notifier dispatches completed jobs to webhook endpoints.
type Notifier interface {
	Notify(ctx context.Context, job *Job) error
}

type HTTPNotifier struct {
	client  *http.Client
	timeout time.Duration
}

func NewHTTPNotifier(client *http.Client, timeout time.Duration) *HTTPNotifier {
	return &HTTPNotifier{client: client, timeout: timeout}
}

func (n *HTTPNotifier) Notify(ctx context.Context, job *Job) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	reqCtx := ctx
	if n.timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, n.timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, job.CallbackURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook unexpected status: %s", resp.Status)
	}
	return nil
}
