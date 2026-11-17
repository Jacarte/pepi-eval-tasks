package billing

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/ids"
)

func TestInvoiceFlow(t *testing.T) {
	ctx := ctxkeys.WithTraceID(context.Background(), "trace-1")
	invoices, charges := NewMemoryInvoiceStore(), NewMemoryChargeStore()
	rec := audit.NewMemory(nil)
	svc := NewService(invoices, charges, rec, nil, ids.NewSequence())
	inv, err := svc.CreateInvoice(ctx, "cus-1", 9900, 3)
	if err != nil {
		t.Fatal(err)
	}
	_ = charges.Put(ctx, &Charge{ID: "ch-1", InvoiceID: inv.ID, Amount: 9900, Status: ChargeStatusAuthorized})
	if _, err := svc.CaptureCharge(ctx, "ch-1", 9900); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.VoidInvoice(ctx, inv.ID); !errors.Is(err, ErrAlreadyPaid) {
		t.Fatalf("err = %v", err)
	}
	if got := len(rec.Events()); got != 3 {
		t.Fatalf("events = %d", got)
	}
	if line := legacyExport(inv); line == "" {
		t.Fatal("empty export")
	}
}
