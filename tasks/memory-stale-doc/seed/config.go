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

	// Backoff sets the waits between attempts.
	Backoff Backoff

	// RetryPolicy is kept for source compatibility.
	//
	// Deprecated: not read since v2.0.0; use Backoff.Kind.
	RetryPolicy string

	// RetryDelay is kept for source compatibility.
	//
	// Deprecated: not read since v2.0.0; use Backoff.Base.
	RetryDelay time.Duration

	// Retryable reports whether an error allows another attempt. The default retries
	// everything except errors wrapping ErrPermanent.
	Retryable func(error) bool

	// Sleep waits for d or until ctx is done. The default uses a timer.
	Sleep func(ctx context.Context, d time.Duration) error
}
