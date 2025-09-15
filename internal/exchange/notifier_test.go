package exchange_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/rate_limiter/internal/exchange"
)

func TestHTTPNotifierSendsPayloadToWebhookClient(t *testing.T) {
	received := make(chan exchange.Job, 1)
	fakeClient := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var job exchange.Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		received <- job
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeClient.Close()

	notifier := exchange.NewHTTPNotifier(fakeClient.Client(), time.Second)
	job := &exchange.Job{ID: "job", Status: exchange.JobStatusCompleted, CallbackURL: fakeClient.URL}

	if err := notifier.Notify(context.Background(), job); err != nil {
		t.Fatalf("notify: %v", err)
	}

	select {
	case got := <-received:
		if got.ID != job.ID {
			t.Fatalf("expected job id %s, got %s", job.ID, got.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive webhook payload")
	}
}
