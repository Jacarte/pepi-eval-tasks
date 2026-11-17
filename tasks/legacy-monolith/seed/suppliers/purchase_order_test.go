package suppliers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func samplePurchaseOrder(n int) *PurchaseOrder {
	return &PurchaseOrder{
		ID:         fmt.Sprintf("purchase_order-%03d", n),
		SupplierID: fmt.Sprintf("supplierid %d;a=b,c\\d", n),
		Total:      money.Cents(n*8 + 174),
		Lines:      n*33 + 174,
		ExpectedAt: time.Date(2023, time.March, n, 9, 30, 0, 0, time.UTC),
		Status:     PurchaseOrderStatusDraft,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestPurchaseOrderCodecRoundTrip(t *testing.T) {
	in := samplePurchaseOrder(4)
	line := EncodePurchaseOrder(in)
	out, err := DecodePurchaseOrder(line)
	if err != nil {
		t.Fatalf("DecodePurchaseOrder(%q): %v", line, err)
	}
	if got := EncodePurchaseOrder(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.SupplierID != in.SupplierID {
		t.Errorf("SupplierID = %q, want %q", out.SupplierID, in.SupplierID)
	}
	if _, err := DecodePurchaseOrder("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestPurchaseOrderStatusTransitions(t *testing.T) {
	for _, s := range []PurchaseOrderStatus{PurchaseOrderStatusDraft, PurchaseOrderStatusCancelled} {
		got, err := ParsePurchaseOrderStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !PurchaseOrderStatusDraft.CanTransition(PurchaseOrderStatusApproved) {
		t.Fatal("first transition should be allowed")
	}
	if PurchaseOrderStatusApproved.CanTransition(PurchaseOrderStatusDraft) {
		t.Fatal("backwards transition allowed")
	}
	if !PurchaseOrderStatusCancelled.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizePurchaseOrders(t *testing.T) {
	items := []*PurchaseOrder{samplePurchaseOrder(1), nil, samplePurchaseOrder(2), samplePurchaseOrder(3)}
	items[2].Status = PurchaseOrderStatusApproved
	sum := SummarizePurchaseOrders(items)
	if sum.Count != 3 || sum.ByStatus[PurchaseOrderStatusDraft] != 2 || sum.ByStatus[PurchaseOrderStatusApproved] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalTotal != 570 {
		t.Errorf("TotalTotal = %v, want 570", sum.TotalTotal)
	}
	if sum.TotalLines != 720 {
		t.Errorf("TotalLines = %v, want 720", sum.TotalLines)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopPurchaseOrdersByTotal(items[2:], 1); len(top) != 1 || top[0].ID != "purchase_order-003" {
		t.Errorf("top = %v", top)
	}
}
