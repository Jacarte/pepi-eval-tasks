package inventory

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

func TestReceiveReserveRelease(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryRepo())
	if _, err := svc.Receive(ctx, "SKU-1", "ams", 10); err != nil {
		t.Fatal(err)
	}
	it, err := svc.Reserve(ctx, "SKU-1", 4)
	if err != nil || it.Reserved != 4 || it.Available() != 6 {
		t.Fatalf("reserve = %+v, %v", it, err)
	}
	if _, err := svc.Reserve(ctx, "SKU-1", 7); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err = %v", err)
	}
	it, err = svc.Release(ctx, "SKU-1", 10)
	if err != nil || it.Reserved != 0 {
		t.Fatalf("release = %+v, %v", it, err)
	}
	if _, err := svc.Reserve(ctx, "SKU-404", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Reserve(ctx, "SKU-1", 0); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("err = %v", err)
	}
}

func TestStockMovementsAreAudited(t *testing.T) {
	rec := audit.NewMemory(nil)
	svc := NewService(NewMemoryRepo(), WithAudit(rec))
	ctx := ctxkeys.WithRequestID(context.Background(), "req-stock")
	if _, err := svc.Receive(ctx, "SKU-5", "rtm", 8); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Release(ctx, "SKU-5", 3); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	if len(evs) != 2 || evs[0].Name != "stock_item.received" || evs[1].Name != "stock_item.released" {
		t.Fatalf("events = %v", evs)
	}
	if evs[0].Subject != "SKU-5" || evs[0].Attrs["warehouse"] != "rtm" || evs[0].Attrs["qty"] != "8" {
		t.Fatalf("received = %v", evs[0])
	}
	// Nothing was reserved, so the release moved zero units.
	if evs[1].Attrs["qty"] != "0" || evs[1].RequestID != "req-stock" {
		t.Fatalf("released = %v", evs[1])
	}
	if _, err := svc.Release(ctx, "SKU-0", 1); errs.Op(err) != "inventory.release" {
		t.Fatalf("op = %q", errs.Op(err))
	}
}
