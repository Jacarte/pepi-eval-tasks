package shipping

import (
	"context"
	"errors"
	"testing"
	"time"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

func newShipping() (*Service, *audit.Memory, *MemoryShipmentStore) {
	clk := clock.NewFixed(time.Date(2024, 4, 2, 8, 0, 0, 0, time.UTC))
	rec := audit.NewMemory(clk)
	store := NewMemoryShipmentStore()
	return NewService(store, rec, clk, ids.NewSequence()), rec, store
}

func TestShipmentLifecycleIsAudited(t *testing.T) {
	svc, rec, _ := newShipping()
	ctx := ctxkeys.WithRequestID(context.Background(), "req-ship-1")
	sh, err := svc.Create(ctx, "ord-1", "ups", 1200)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Hold(ctx, sh.ID, "address check"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Release(ctx, sh.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Dispatch(ctx, sh.ID, "1Z999"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Deliver(ctx, sh.ID); err != nil {
		t.Fatal(err)
	}
	want := []string{"shipment.created", "shipment.held", "shipment.released", "shipment.dispatched", "shipment.delivered"}
	evs := rec.Events()
	if len(evs) != len(want) {
		t.Fatalf("got %d events: %v", len(evs), evs)
	}
	for i, ev := range evs {
		if ev.Name != want[i] || ev.Subject != sh.ID || ev.RequestID != "req-ship-1" {
			t.Errorf("event %d = %+v", i, ev)
		}
	}
}

func TestShippingErrors(t *testing.T) {
	svc, rec, _ := newShipping()
	ctx := context.Background()
	_, err := svc.Create(ctx, "", "ups", 1)
	if !errors.Is(err, ErrMissingOrder) || errs.Op(err) != "shipping.create" {
		t.Fatalf("err = %v", err)
	}
	_, err = svc.Deliver(ctx, "shp-404")
	if !errors.Is(err, ErrNotFound) || errs.Op(err) != "shipping.deliver" {
		t.Fatalf("err = %v", err)
	}
	if n := len(rec.Events()); n != 0 {
		t.Fatalf("failed calls produced %d audit events", n)
	}
}

func TestQuote(t *testing.T) {
	c, err := Quote("UPS", ZoneFor("us"), 400)
	if err != nil || c <= 0 {
		t.Fatalf("Quote = %v, %v", c, err)
	}
	if _, err := Quote("ups", 1, 1_000_000); !errors.Is(err, ErrNoRate) {
		t.Fatalf("err = %v", err)
	}
	if ZoneFor("BR") != 4 {
		t.Fatal("unknown countries should be zone 4")
	}
}
