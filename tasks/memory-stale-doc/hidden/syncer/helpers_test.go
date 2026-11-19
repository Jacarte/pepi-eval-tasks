package syncer_test

import (
	"context"
	"errors"
	"time"

	"jobq"
)

var errBoom = errors.New("boom")

// drive runs an always-failing job (or one returning fail) under cfg with a recording Sleep and
// returns the attempt count, the recorded waits and Run's error.
func drive(cfg jobq.Config, fail error) (int, []time.Duration, error) {
	var waits []time.Duration
	cfg.Sleep = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	calls := 0
	err := jobq.New(cfg).Run(context.Background(), func(context.Context) error {
		calls++
		return fail
	})
	return calls, waits, err
}
