package exchange

import "time"

// Rate represents a currency exchange rate for a specific base/quote pair.
type Rate struct {
	Base  string    `json:"base"`
	Quote string    `json:"quote"`
	Value float64   `json:"value"`
	AsOf  time.Time `json:"as_of"`
}

// JobStatus describes the state of an asynchronous exchange rate request.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job stores the status of asynchronous rate retrieval.
type Job struct {
	ID          string    `json:"id"`
	Status      JobStatus `json:"status"`
	Rate        *Rate     `json:"rate,omitempty"`
	Failure     string    `json:"failure,omitempty"`
	CallbackURL string    `json:"callback_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
