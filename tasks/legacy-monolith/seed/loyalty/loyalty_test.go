package loyalty

import (
	"context"
	"errors"
	"testing"
)

func TestAccrueRedeem(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryRepo(), nil, nil)
	a, err := svc.Open(ctx, "cus-9")
	if err != nil || a.ID != "loy-cus-9" || a.TierName != "bronze" {
		t.Fatalf("open = %+v, %v", a, err)
	}
	if a, err = svc.Accrue(ctx, a.ID, 12000); err != nil || a.TierName != "silver" {
		t.Fatalf("accrue = %+v, %v", a, err)
	}
	if a, err = svc.Redeem(ctx, a.ID, 2000); err != nil || a.Balance != 10000 || a.Lifetime != 12000 {
		t.Fatalf("redeem = %+v, %v", a, err)
	}
	if _, err := svc.Redeem(ctx, a.ID, 10001); !errors.Is(err, ErrInsufficientPoints) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Redeem(ctx, a.ID, -1); !errors.Is(err, ErrInvalidPoints) {
		t.Fatalf("err = %v", err)
	}
	if again, _ := svc.Open(ctx, "cus-9"); again.Balance != 10000 {
		t.Fatal("Open is not idempotent")
	}
}
