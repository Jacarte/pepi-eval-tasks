package jobq

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func record(waits *[]time.Duration) func(context.Context, time.Duration) error {
	return func(_ context.Context, d time.Duration) error {
		*waits = append(*waits, d)
		return nil
	}
}

func TestRunRetriesUntilSuccess(t *testing.T) {
	var waits []time.Duration
	r := New(Config{MaxAttempts: 4, Backoff: Backoff{Kind: Linear, Base: 10 * time.Millisecond}, Sleep: record(&waits)})
	calls := 0
	err := r.Run(context.Background(), func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("flaky")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
	want := []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("waits %v, want %v", waits, want)
	}
}

func TestRunStopsOnPermanent(t *testing.T) {
	var waits []time.Duration
	r := New(Config{MaxAttempts: 4, Sleep: record(&waits)})
	calls := 0
	err := r.Run(context.Background(), func(context.Context) error {
		calls++
		return Permanent(errors.New("bad input"))
	})
	if !errors.Is(err, ErrPermanent) || calls != 1 || len(waits) != 0 {
		t.Fatalf("err=%v calls=%d waits=%v", err, calls, waits)
	}
}

func TestBackoffWaits(t *testing.T) {
	ms := time.Millisecond
	cases := []struct {
		b    Backoff
		want []time.Duration
	}{
		{Backoff{}, []time.Duration{time.Second, time.Second, time.Second}},
		{Backoff{Kind: Constant, Base: 5 * ms}, []time.Duration{5 * ms, 5 * ms, 5 * ms}},
		{Backoff{Kind: Linear, Base: 5 * ms, Max: 12 * ms}, []time.Duration{5 * ms, 10 * ms, 12 * ms}},
		{Backoff{Kind: Exponential, Base: 5 * ms, Max: 15 * ms}, []time.Duration{5 * ms, 10 * ms, 15 * ms}},
	}
	for _, c := range cases {
		var got []time.Duration
		for n := 1; n <= 3; n++ {
			got = append(got, c.b.wait(n))
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v: waits %v, want %v", c.b, got, c.want)
		}
	}
}
