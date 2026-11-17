package billing

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleCharge(n int) *Charge {
	return &Charge{
		ID:        fmt.Sprintf("charge-%03d", n),
		InvoiceID: fmt.Sprintf("invoiceid %d;a=b,c\\d", n),
		Amount:    money.Cents(n*24 + 297),
		Captured:  money.Cents(n*4 + 397),
		Processor: fmt.Sprintf("processor-%d", n),
		Status:    ChargeStatusPending,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestChargeValidate(t *testing.T) {
	if err := sampleCharge(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleCharge(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleCharge(1)
	bad.InvoiceID = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing invoice id: err = %v", err)
	}
	bad = sampleCharge(1)
	bad.Status = ChargeStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestChargeStatusTransitions(t *testing.T) {
	for _, s := range []ChargeStatus{ChargeStatusPending, ChargeStatusFailed} {
		got, err := ParseChargeStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ChargeStatusPending.CanTransition(ChargeStatusAuthorized) {
		t.Fatal("first transition should be allowed")
	}
	if ChargeStatusAuthorized.CanTransition(ChargeStatusPending) {
		t.Fatal("backwards transition allowed")
	}
	if !ChargeStatusFailed.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeCharges(t *testing.T) {
	items := []*Charge{sampleCharge(1), nil, sampleCharge(2), sampleCharge(3)}
	items[2].Status = ChargeStatusAuthorized
	sum := SummarizeCharges(items)
	if sum.Count != 3 || sum.ByStatus[ChargeStatusPending] != 2 || sum.ByStatus[ChargeStatusAuthorized] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalAmount != 1035 {
		t.Errorf("TotalAmount = %v, want 1035", sum.TotalAmount)
	}
	if sum.TotalCaptured != 1215 {
		t.Errorf("TotalCaptured = %v, want 1215", sum.TotalCaptured)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopChargesByAmount(items[2:], 1); len(top) != 1 || top[0].ID != "charge-003" {
		t.Errorf("top = %v", top)
	}
}
