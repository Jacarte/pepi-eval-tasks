package warehouse

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

func TestAllocateAndBlock(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-wh")
	bins := NewMemoryBinSlotStore()
	rec := audit.NewMemory(nil)
	svc := NewService(bins, NewMemoryTransferStore(), rec, nil, ids.NewSequence())
	_ = bins.Store(ctx, &BinSlot{ID: "A-01-1", Zone: "A", Aisle: 1, Level: 1, Capacity: 40, Status: BinSlotStatusFree})
	_ = bins.Store(ctx, &BinSlot{ID: "A-01-2", Zone: "A", Aisle: 1, Level: 2, Capacity: 10, Status: BinSlotStatusFree})

	if _, err := svc.Allocate(ctx, "A-01-1", "SKU-7"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Allocate(ctx, "A-01-1", "SKU-8"); !errors.Is(err, ErrBinNotFree) || errs.Op(err) != "warehouse.allocate" {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.PlanTransfer(ctx, "A-01-1", "A-01-2", "SKU-7", 11); !errors.Is(err, ErrOverCapacity) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Block(ctx, "A-01-2"); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	if len(evs) != 2 || evs[0].Name != "bin_slot.allocated" || evs[1].Name != "bin_slot.blocked" || evs[1].Subject != "A-01-2" {
		t.Fatalf("events = %v", evs)
	}
}
