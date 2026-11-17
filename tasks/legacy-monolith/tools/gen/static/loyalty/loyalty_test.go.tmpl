package loyalty

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/errs"
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

func TestAccrueIsAuditedUnderCustomer(t *testing.T) {
	rec := audit.NewMemory(nil)
	repo := NewMemoryRepo()
	svc := NewService(repo, rec, nil)
	ctx := ctxkeys.WithRequestID(context.Background(), "req-loy")
	_ = repo.Store(ctx, &Account{ID: "acct-merged-2", CustomerID: "cus-4", TierName: "bronze"})
	if _, err := svc.Accrue(ctx, "acct-merged-2", 250); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	if len(evs) != 1 || evs[0].Name != "account.accrued" || evs[0].Subject != "cus-4" || evs[0].RequestID != "req-loy" {
		t.Fatalf("events = %v", evs)
	}
	if evs[0].Attrs["account_id"] != "acct-merged-2" || evs[0].Attrs["points"] != "250" {
		t.Fatalf("attrs = %v", evs[0].Attrs)
	}
	if _, err := svc.Accrue(ctx, "acct-merged-2", 0); errs.Op(err) != "loyalty.accrue" {
		t.Fatalf("op = %q", errs.Op(err))
	}
}
