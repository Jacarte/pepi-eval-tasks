package coupons

import (
	"context"
	"errors"
	"testing"
	"time"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

func TestActivateRedeem(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-cpn")
	clk := clock.NewFixed(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))
	store := NewMemoryCouponStore()
	rec := audit.NewMemory(clk)
	svc := NewService(store, rec, clk)
	_ = store.Store(ctx, &Coupon{ID: "cpn-1", Code: "summer10", PercentOff: 10, MaxUses: 2, Status: CouponStatusDraft, ExpiresAt: time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)})
	if _, err := svc.Redeem(ctx, "cpn-1"); !errors.Is(err, ErrNotActive) || errs.Op(err) != "coupons.redeem" {
		t.Fatalf("err = %v", err)
	}
	c, err := svc.Activate(ctx, "cpn-1")
	if err != nil || c.Code != "SUMMER10" {
		t.Fatalf("activate = %+v, %v", c, err)
	}
	for i := 0; i < 2; i++ {
		if _, err := svc.Redeem(ctx, "cpn-1"); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := store.Load(ctx, "cpn-1")
	if got.Status != CouponStatusExhausted {
		t.Fatalf("status = %v", got.Status)
	}
	if n := len(rec.Named("coupon.redeemed")); n != 2 {
		t.Fatalf("redeemed events = %d", n)
	}
}
