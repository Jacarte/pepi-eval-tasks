package payments

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleSettlement(n int) *Settlement {
	return &Settlement{
		ID:        fmt.Sprintf("settlement-%03d", n),
		Batch:     fmt.Sprintf("batch %d;a=b,c\\d", n),
		Amount:    money.Cents(n*32 + 449),
		Processor: fmt.Sprintf("processor %d;a=b,c\\d", n),
		Items:     n*30 + 409,
		Status:    SettlementStatusOpen,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestSettlementCodecRoundTrip(t *testing.T) {
	in := sampleSettlement(4)
	line := EncodeSettlement(in)
	out, err := DecodeSettlement(line)
	if err != nil {
		t.Fatalf("DecodeSettlement(%q): %v", line, err)
	}
	if got := EncodeSettlement(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Batch != in.Batch {
		t.Errorf("Batch = %q, want %q", out.Batch, in.Batch)
	}
	if out.Processor != in.Processor {
		t.Errorf("Processor = %q, want %q", out.Processor, in.Processor)
	}
	if _, err := DecodeSettlement("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestSettlementValidate(t *testing.T) {
	if err := sampleSettlement(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleSettlement(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleSettlement(1)
	bad.Batch = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing batch: err = %v", err)
	}
	bad = sampleSettlement(1)
	bad.Status = SettlementStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestSettlementStatusTransitions(t *testing.T) {
	for _, s := range []SettlementStatus{SettlementStatusOpen, SettlementStatusSettled} {
		got, err := ParseSettlementStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !SettlementStatusOpen.CanTransition(SettlementStatusSubmitted) {
		t.Fatal("first transition should be allowed")
	}
	if SettlementStatusSubmitted.CanTransition(SettlementStatusOpen) {
		t.Fatal("backwards transition allowed")
	}
	if !SettlementStatusSettled.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
