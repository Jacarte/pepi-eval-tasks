package orders

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

func newTestService() (*Service, *MemoryRepo) {
	repo := NewMemoryRepo()
	svc := NewService(Config{
		Repo:  repo,
		Clock: clock.NewFixed(time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)),
		IDs:   ids.NewSequence(),
	})
	return svc, repo
}

func TestPlacePricesOrder(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	o, err := svc.Place(ctx, PlaceRequest{
		CustomerID: "cus-1",
		Region:     "us-ca",
		Lines:      []Line{{SKU: "A", Qty: 2, UnitPrice: 1000}, {SKU: "B", Qty: 1, UnitPrice: 500}},
		PercentOff: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.ID != "ord-000001" || o.Subtotal != 2500 || o.Discount != 250 || o.Tax != 163 || o.Total != 2413 {
		t.Fatalf("order = %+v", o)
	}
	stored, err := repo.Get(ctx, o.ID)
	if err != nil || stored.Status != StatusPlaced || stored.ItemCount() != 3 {
		t.Fatalf("stored = %+v, %v", stored, err)
	}
}

func TestPlaceValidation(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	cases := []struct {
		req  PlaceRequest
		want error
	}{
		{PlaceRequest{Region: "US-CA", Lines: []Line{{SKU: "A", Qty: 1}}}, ErrMissingCustomer},
		{PlaceRequest{CustomerID: "c", Region: "US-CA"}, ErrEmptyOrder},
		{PlaceRequest{CustomerID: "c", Region: "US-CA", Lines: []Line{{SKU: "A", Qty: 0}}}, ErrInvalidQuantity},
	}
	for _, tc := range cases {
		if _, err := svc.Place(ctx, tc.req); !errors.Is(err, tc.want) {
			t.Errorf("Place(%+v) err = %v, want %v", tc.req, err, tc.want)
		}
	}
	if got, _ := repo.ListByCustomer(ctx, "c"); len(got) != 0 {
		t.Fatal("invalid orders were stored")
	}
}

func TestCancel(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	o, err := svc.Place(ctx, PlaceRequest{CustomerID: "c", Region: "DE", Lines: []Line{{SKU: "A", Qty: 1, UnitPrice: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Cancel(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Cancel(ctx, o.ID); !errors.Is(err, ErrNotCancellable) {
		t.Fatalf("second cancel err = %v", err)
	}
	if _, err := svc.Cancel(ctx, "ord-404"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing order err = %v", err)
	}
}

func TestCancelIsAudited(t *testing.T) {
	rec := audit.NewMemory(nil)
	svc := NewService(Config{Repo: NewMemoryRepo(), Audit: rec, IDs: ids.NewSequence()})
	o, err := svc.Place(context.Background(), PlaceRequest{CustomerID: "cus-9", Region: "DE", Lines: []Line{{SKU: "A", Qty: 3, UnitPrice: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	rec.Reset()
	ctx := ctxkeys.WithRequestID(context.Background(), "req-cancel")
	if _, err := svc.Cancel(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Cancel(ctx, o.ID); errs.Op(err) != "orders.cancel" {
		t.Fatalf("second cancel op = %q", errs.Op(err))
	}
	evs := rec.Named("order.cancelled")
	if len(evs) != 1 || evs[0].Subject != o.ID || evs[0].RequestID != "req-cancel" || evs[0].Attrs["customer_id"] != "cus-9" {
		t.Fatalf("events = %v", rec.Events())
	}
	if err := svc.MarkFulfilled(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	if evs := rec.Named("order.fulfilled"); len(evs) != 1 || evs[0].Attrs["customer_id"] != "cus-9" || evs[0].Attrs["items"] != "3" {
		t.Fatalf("events = %v", rec.Events())
	}
}
