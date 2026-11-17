package suppliers

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

func TestOnboardApproveReceive(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-sup")
	sups, pos := NewMemorySupplierStore(), NewMemoryPurchaseOrderStore()
	rec := audit.NewMemory(nil)
	svc := NewService(sups, pos, rec, nil)
	_ = sups.Save(ctx, &Supplier{ID: "sup-1", Name: "Acme", Country: "NL", Status: SupplierStatusProspect})
	_ = pos.Save(ctx, &PurchaseOrder{ID: "po-1", SupplierID: "sup-1", Total: 120000, Lines: 3, Status: PurchaseOrderStatusDraft})

	if _, err := svc.Approve(ctx, "po-1"); !errors.Is(err, ErrNotOnboarded) || errs.Op(err) != "suppliers.approve" {
		t.Fatalf("approve before onboarding: %v", err)
	}
	if _, err := svc.Onboard(ctx, "sup-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(ctx, "po-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Receive(ctx, "po-1"); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	names := []string{"supplier.onboarded", "purchase_order.approved", "purchase_order.received"}
	if len(evs) != 3 {
		t.Fatalf("events = %v", evs)
	}
	for i, n := range names {
		if evs[i].Name != n || evs[i].RequestID != "req-sup" {
			t.Errorf("event %d = %+v", i, evs[i])
		}
	}
	card, err := svc.Scorecard(ctx, "sup-1")
	if err != nil || card.Count != 1 || card.TotalTotal != 120000 {
		t.Fatalf("scorecard = %+v, %v", card, err)
	}
}
