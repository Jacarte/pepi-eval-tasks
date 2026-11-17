package scheduling

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleSlot(n int) *Slot {
	return &Slot{
		ID:        fmt.Sprintf("slot-%03d", n),
		Warehouse: fmt.Sprintf("warehouse %d;a=b,c\\d", n),
		Window:    fmt.Sprintf("window-%d", n),
		Capacity:  n*24 + 375,
		Booked:    n*26 + 326,
		Status:    SlotStatusOpen,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestSlotCodecRoundTrip(t *testing.T) {
	in := sampleSlot(4)
	line := EncodeSlot(in)
	out, err := DecodeSlot(line)
	if err != nil {
		t.Fatalf("DecodeSlot(%q): %v", line, err)
	}
	if got := EncodeSlot(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Warehouse != in.Warehouse {
		t.Errorf("Warehouse = %q, want %q", out.Warehouse, in.Warehouse)
	}
	if out.Window != in.Window {
		t.Errorf("Window = %q, want %q", out.Window, in.Window)
	}
	if _, err := DecodeSlot("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemorySlotStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemorySlotStore()
	for n := 1; n <= 3; n++ {
		if err := st.Store(ctx, sampleSlot(n)); err != nil {
			t.Fatalf("Store(%d): %v", n, err)
		}
	}
	got, err := st.Load(ctx, "slot-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = SlotStatusClosed
	again, _ := st.Load(ctx, "slot-002")
	if again.Status != SlotStatusOpen {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Store(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.Scan(ctx, SlotFilter{Status: SlotStatusOpen})
	if err != nil || len(list) != 2 || list[0].ID != "slot-001" {
		t.Fatalf("Scan by status = %v, %v", list, err)
	}
	if err := st.Drop(ctx, "slot-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(ctx, "slot-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestSlotStatusTransitions(t *testing.T) {
	for _, s := range []SlotStatus{SlotStatusOpen, SlotStatusClosed} {
		got, err := ParseSlotStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !SlotStatusOpen.CanTransition(SlotStatusFull) {
		t.Fatal("first transition should be allowed")
	}
	if SlotStatusFull.CanTransition(SlotStatusOpen) {
		t.Fatal("backwards transition allowed")
	}
	if !SlotStatusClosed.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
