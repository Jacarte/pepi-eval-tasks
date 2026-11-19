package jobq

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// The v1 fields still compile but have no effect: the waits stay at DefaultBackoff.
func TestLegacyRetryFieldsAreIgnored(t *testing.T) {
	var waits []time.Duration
	r := New(Config{
		MaxAttempts: 3,
		RetryPolicy: "linear",
		RetryDelay:  250 * time.Millisecond,
		Sleep:       record(&waits),
	})
	_ = r.Run(context.Background(), func(context.Context) error { return errors.New("down") })
	want := []time.Duration{time.Second, time.Second}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("waits %v, want %v", waits, want)
	}
}
