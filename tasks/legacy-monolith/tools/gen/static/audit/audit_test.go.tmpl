package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"shopcore/clock"
)

func TestMemoryStampsAndCopies(t *testing.T) {
	at := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	m := NewMemory(clock.NewFixed(at))
	attrs := map[string]string{"k": "v"}
	if err := m.Record(context.Background(), Event{Name: "thing.done", Subject: "t-1", Attrs: attrs}); err != nil {
		t.Fatal(err)
	}
	attrs["k"] = "changed"
	evs := m.Events()
	if len(evs) != 1 || !evs[0].At.Equal(at) || evs[0].Attrs["k"] != "v" {
		t.Fatalf("events = %+v", evs)
	}
	if len(m.Named("thing.done")) != 1 || len(m.ForSubject("t-2")) != 0 {
		t.Fatal("filters broken")
	}
	m.Reset()
	if len(m.Events()) != 0 {
		t.Fatal("reset did not clear")
	}
}

func TestMemoryRejects(t *testing.T) {
	m := NewMemory(nil)
	if err := m.Record(context.Background(), Event{}); !errors.Is(err, ErrMissingName) {
		t.Fatalf("err = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Record(ctx, Event{Name: "x.y"}); err == nil {
		t.Fatal("cancelled context accepted")
	}
}

func TestMultiStopsOnError(t *testing.T) {
	boom := errors.New("boom")
	var calls int
	mr := Multi{RecorderFunc(func(context.Context, Event) error { calls++; return boom }), RecorderFunc(func(context.Context, Event) error { calls++; return nil })}
	if err := mr.Record(context.Background(), Event{Name: "a.b"}); !errors.Is(err, boom) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
	if OrDiscard(nil).Record(context.Background(), Event{}) != nil {
		t.Fatal("discard should never fail")
	}
}

func TestEventString(t *testing.T) {
	ev := Event{Name: "a.b", Subject: "s", RequestID: "r", Attrs: map[string]string{"z": "1", "a": "2"}}
	if got := ev.String(); got != "a.b subject=s request=r a=2 z=1" {
		t.Fatalf("String = %q", got)
	}
}
