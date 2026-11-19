package jobq

import (
	"context"
	"time"
)

// Config configures a Runner.
type Config struct {
	// Name labels the job in error messages.
	Name string

	// MaxAttempts is the number of tries, the first one included. 0 means 3.
	MaxAttempts int

	// RetryPolicy names the schedule.
	RetryPolicy string

	// RetryDelay is the first wait.
	RetryDelay time.Duration

	// RetryMaxDelay caps every wait.
	RetryMaxDelay time.Duration

	// Backoff sets the waits between attempts.
	Backoff Backoff

	// Retryable reports whether an error allows another attempt. The default retries
	// everything except errors wrapping ErrPermanent.
	Retryable func(error) bool

	// Sleep waits for d or until ctx is done. The default uses a timer.
	Sleep func(ctx context.Context, d time.Duration) error
}
