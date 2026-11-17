package warehouse

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleBinSlot(n int) *BinSlot {
	return &BinSlot{
		ID:        fmt.Sprintf("bin_slot-%03d", n),
		Zone:      fmt.Sprintf("zone %d;a=b,c\\d", n),
		Aisle:     n*3 + 263,
		Level:     n*4 + 266,
		SKU:       fmt.Sprintf("sku-%d", n),
		Capacity:  n*12 + 180,
		Status:    BinSlotStatusFree,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestBinSlotCodecRoundTrip(t *testing.T) {
	in := sampleBinSlot(4)
	line := EncodeBinSlot(in)
	out, err := DecodeBinSlot(line)
	if err != nil {
		t.Fatalf("DecodeBinSlot(%q): %v", line, err)
	}
	if got := EncodeBinSlot(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Zone != in.Zone {
		t.Errorf("Zone = %q, want %q", out.Zone, in.Zone)
	}
	if out.SKU != in.SKU {
		t.Errorf("SKU = %q, want %q", out.SKU, in.SKU)
	}
	if _, err := DecodeBinSlot("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryBinSlotStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryBinSlotStore()
	for n := 1; n <= 3; n++ {
		if err := st.Store(ctx, sampleBinSlot(n)); err != nil {
			t.Fatalf("Store(%d): %v", n, err)
		}
	}
	got, err := st.Load(ctx, "bin_slot-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = BinSlotStatusBlocked
	again, _ := st.Load(ctx, "bin_slot-002")
	if again.Status != BinSlotStatusFree {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Store(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.Scan(ctx, BinSlotFilter{Status: BinSlotStatusFree})
	if err != nil || len(list) != 2 || list[0].ID != "bin_slot-001" {
		t.Fatalf("Scan by status = %v, %v", list, err)
	}
	if err := st.Drop(ctx, "bin_slot-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(ctx, "bin_slot-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}
