// Package jobq runs a job and retries it with waits between attempts.
package jobq

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Runner runs jobs with the retry rules of its Config.
type Runner struct{ cfg Config }

// New returns a Runner for cfg, filling in defaults.
func New(cfg Config) *Runner {
	if cfg.MaxAttempts == 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.Retryable == nil {
		cfg.Retryable = func(err error) bool { return !errors.Is(err, ErrPermanent) }
	}
	if cfg.Sleep == nil {
		cfg.Sleep = sleep
	}
	return &Runner{cfg: cfg}
}

// Run calls job until it succeeds, fails with an error that is not retryable, or the attempts
// run out. A Sleep error, such as a cancelled context, is returned as is.
func (r *Runner) Run(ctx context.Context, job func(ctx context.Context) error) error {
	var err error
	for attempt := 1; attempt <= r.cfg.MaxAttempts; attempt++ {
		if err = job(ctx); err == nil {
			return nil
		}
		if attempt == r.cfg.MaxAttempts || !r.cfg.Retryable(err) {
			break
		}
		if serr := r.cfg.Sleep(ctx, r.cfg.Backoff.wait(attempt)); serr != nil {
			return serr
		}
	}
	return fmt.Errorf("%s: %w", r.cfg.Name, err)
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
