package billing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleInvoice(n int) *Invoice {
	return &Invoice{
		ID:         fmt.Sprintf("invoice-%03d", n),
		CustomerID: fmt.Sprintf("customerid %d;a=b,c\\d", n),
		Number:     fmt.Sprintf("number-%d", n),
		AmountDue:  money.Cents(n*21 + 342),
		Currency:   fmt.Sprintf("currency-%d", n),
		DueAt:      time.Date(2023, time.March, n, 9, 30, 0, 0, time.UTC),
		Lines:      n*23 + 125,
		Status:     InvoiceStatusDraft,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestInvoiceCodecRoundTrip(t *testing.T) {
	in := sampleInvoice(4)
	line := EncodeInvoice(in)
	out, err := DecodeInvoice(line)
	if err != nil {
		t.Fatalf("DecodeInvoice(%q): %v", line, err)
	}
	if got := EncodeInvoice(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.CustomerID != in.CustomerID {
		t.Errorf("CustomerID = %q, want %q", out.CustomerID, in.CustomerID)
	}
	if out.Number != in.Number {
		t.Errorf("Number = %q, want %q", out.Number, in.Number)
	}
	if out.Currency != in.Currency {
		t.Errorf("Currency = %q, want %q", out.Currency, in.Currency)
	}
	if _, err := DecodeInvoice("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryInvoiceStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryInvoiceStore()
	for n := 1; n <= 3; n++ {
		if err := st.Put(ctx, sampleInvoice(n)); err != nil {
			t.Fatalf("Put(%d): %v", n, err)
		}
	}
	got, err := st.Get(ctx, "invoice-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = InvoiceStatusVoid
	again, _ := st.Get(ctx, "invoice-002")
	if again.Status != InvoiceStatusDraft {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Put(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.List(ctx, InvoiceFilter{Status: InvoiceStatusDraft})
	if err != nil || len(list) != 2 || list[0].ID != "invoice-001" {
		t.Fatalf("List by status = %v, %v", list, err)
	}
	if err := st.Delete(ctx, "invoice-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, "invoice-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestSummarizeInvoices(t *testing.T) {
	items := []*Invoice{sampleInvoice(1), nil, sampleInvoice(2), sampleInvoice(3)}
	items[2].Status = InvoiceStatusOpen
	sum := SummarizeInvoices(items)
	if sum.Count != 3 || sum.ByStatus[InvoiceStatusDraft] != 2 || sum.ByStatus[InvoiceStatusOpen] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalAmountDue != 1152 {
		t.Errorf("TotalAmountDue = %v, want 1152", sum.TotalAmountDue)
	}
	if sum.TotalLines != 513 {
		t.Errorf("TotalLines = %v, want 513", sum.TotalLines)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopInvoicesByAmountDue(items[2:], 1); len(top) != 1 || top[0].ID != "invoice-003" {
		t.Errorf("top = %v", top)
	}
}
