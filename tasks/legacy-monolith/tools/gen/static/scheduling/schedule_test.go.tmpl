package scheduling

import (
	"context"
	"errors"
	"testing"
	"time"

	"shopcore/clock"
	"shopcore/ctxkeys"
)

func TestParseSpecNext(t *testing.T) {
	sp, err := ParseSpec("daily 02:30")
	if err != nil {
		t.Fatal(err)
	}
	last := time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC)
	if got := sp.Next(last); !got.Equal(time.Date(2024, 1, 2, 2, 30, 0, 0, time.UTC)) {
		t.Fatalf("Next = %v", got)
	}
	sp, _ = ParseSpec("every 15m")
	if got := sp.Next(last); !got.Equal(last.Add(15 * time.Minute)) {
		t.Fatalf("Next = %v", got)
	}
	for _, bad := range []string{"every 1s", "daily 25:00", "weekly"} {
		if _, err := ParseSpec(bad); !errors.Is(err, ErrBadSpec) {
			t.Errorf("ParseSpec(%q) err = %v", bad, err)
		}
	}
}

func TestRunnerRunsDueJobsWithRequestIDs(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewFixed(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	store := NewMemoryJobStore()
	_ = store.Store(ctx, &Job{ID: "j1", Name: "exports", Spec: "hourly", Status: JobStatusIdle})
	_ = store.Store(ctx, &Job{ID: "j2", Name: "cleanup", Spec: "every 30m", Status: JobStatusIdle, LastRunAt: clk.Now().Add(-10 * time.Minute)})
	r := NewRunner(store, clk)
	var seen []string
	r.Register("exports", func(ctx context.Context) error {
		seen = append(seen, ctxkeys.RequestIDFrom(ctx)+"/"+ctxkeys.ActorFrom(ctx))
		return nil
	})
	r.Register("cleanup", func(context.Context) error { return nil })
	ran, err := r.RunDue(ctx)
	if err != nil || len(ran) != 1 || ran[0] != "exports" {
		t.Fatalf("ran = %v, %v", ran, err)
	}
	if len(seen) != 1 || seen[0] != "job-exports-1/scheduler" {
		t.Fatalf("seen = %v", seen)
	}
	clk.Advance(time.Hour)
	ran, _ = r.RunDue(ctx)
	if len(ran) != 2 {
		t.Fatalf("second pass ran %v", ran)
	}
}

func TestBook(t *testing.T) {
	ctx := context.Background()
	slots := NewMemorySlotStore()
	_ = slots.Store(ctx, &Slot{ID: "s1", Warehouse: "ams", Window: "09:00-12:00", Capacity: 1, Status: SlotStatusOpen})
	if _, err := Book(ctx, slots, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Book(ctx, slots, "s1"); !errors.Is(err, ErrSlotFull) {
		t.Fatalf("err = %v", err)
	}
}
