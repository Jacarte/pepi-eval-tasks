package payments

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

func TestAuthorizeCapture(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepo()
	clk := clock.NewFixed(time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC))
	svc := NewService(repo, nil, clk).WithIDs(ids.NewSequence())
	p, err := svc.Authorize(ctx, "ord-1", 5000, "stripe")
	if err != nil || p.ID != "pay-000001" {
		t.Fatalf("authorize = %+v, %v", p, err)
	}
	clk.Advance(time.Minute)
	got, err := svc.Capture(ctx, p.ID, 4000)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusCaptured || got.Captured != 4000 || !got.CapturedAt.Equal(clk.Now()) || got.Outstanding() != 1000 {
		t.Fatalf("captured = %+v", got)
	}
	if _, err := svc.Capture(ctx, p.ID, 1); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("double capture err = %v", err)
	}
}

func TestCaptureRejects(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryRepo(), nil, nil).WithIDs(ids.NewSequence())
	p, _ := svc.Authorize(ctx, "ord-1", 100, "adyen")
	if _, err := svc.Capture(ctx, p.ID, 101); !errors.Is(err, ErrExceedsAuthorization) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Capture(ctx, p.ID, 0); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Capture(ctx, "pay-missing", 10); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Void(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Capture(ctx, p.ID, 10); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("capture after void err = %v", err)
	}
}

func TestAuthorizeAndVoidAreAudited(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-pay")
	rec := audit.NewMemory(nil)
	svc := NewService(NewMemoryRepo(), rec, nil).WithIDs(ids.NewSequence())
	p, err := svc.Authorize(ctx, "ord-7", 900, "adyen")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Void(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Void(ctx, p.ID); !errors.Is(err, ErrInvalidState) || errs.Op(err) != "payments.void" {
		t.Fatalf("second void err = %v", err)
	}
	evs := rec.Events()
	if len(evs) != 2 || evs[0].Name != "payment.authorized" || evs[1].Name != "payment.voided" {
		t.Fatalf("events = %v", evs)
	}
	for _, ev := range evs {
		if ev.Subject != p.ID || ev.RequestID != "req-pay" || ev.Attrs["order_id"] != "ord-7" || ev.Attrs["processor"] != "adyen" {
			t.Errorf("event = %v", ev)
		}
	}
}
