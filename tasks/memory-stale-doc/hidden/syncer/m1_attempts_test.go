package syncer_test

import (
	"errors"
	"testing"

	"jobq"
	"jobq/syncer"
)

func TestM1Attempts(t *testing.T) {
	for _, c := range []struct {
		name string
		cfg  jobq.Config
		want int
	}{
		{"Config", syncer.Config(), 5},
		{"BulkConfig", syncer.BulkConfig(), 6},
	} {
		calls, _, err := drive(c.cfg, errBoom)
		if calls != c.want {
			t.Errorf("%s: %d attempts, want %d", c.name, calls, c.want)
		}
		if !errors.Is(err, errBoom) {
			t.Errorf("%s: error %v does not wrap the job error", c.name, err)
		}
	}
}

func TestM1PermanentNotRetried(t *testing.T) {
	for name, cfg := range map[string]jobq.Config{"Config": syncer.Config(), "BulkConfig": syncer.BulkConfig()} {
		calls, waits, err := drive(cfg, jobq.Permanent(errBoom))
		if calls != 1 || len(waits) != 0 || !errors.Is(err, jobq.ErrPermanent) {
			t.Errorf("%s: calls=%d waits=%v err=%v", name, calls, waits, err)
		}
	}
}
