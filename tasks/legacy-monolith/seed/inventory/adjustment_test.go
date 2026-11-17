package inventory

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleAdjustment(n int) *Adjustment {
	return &Adjustment{
		ID:        fmt.Sprintf("adjustment-%03d", n),
		SKU:       fmt.Sprintf("sku %d;a=b,c\\d", n),
		Delta:     n*20 + 438,
		Reason:    fmt.Sprintf("reason %d;a=b,c\\d", n),
		Operator:  fmt.Sprintf("operator-%d", n),
		Status:    AdjustmentStatusPending,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestAdjustmentCodecRoundTrip(t *testing.T) {
	in := sampleAdjustment(4)
	line := EncodeAdjustment(in)
	out, err := DecodeAdjustment(line)
	if err != nil {
		t.Fatalf("DecodeAdjustment(%q): %v", line, err)
	}
	if got := EncodeAdjustment(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.SKU != in.SKU {
		t.Errorf("SKU = %q, want %q", out.SKU, in.SKU)
	}
	if out.Reason != in.Reason {
		t.Errorf("Reason = %q, want %q", out.Reason, in.Reason)
	}
	if out.Operator != in.Operator {
		t.Errorf("Operator = %q, want %q", out.Operator, in.Operator)
	}
	if _, err := DecodeAdjustment("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestAdjustmentStatusTransitions(t *testing.T) {
	for _, s := range []AdjustmentStatus{AdjustmentStatusPending, AdjustmentStatusRejected} {
		got, err := ParseAdjustmentStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !AdjustmentStatusPending.CanTransition(AdjustmentStatusApplied) {
		t.Fatal("first transition should be allowed")
	}
	if AdjustmentStatusApplied.CanTransition(AdjustmentStatusPending) {
		t.Fatal("backwards transition allowed")
	}
	if !AdjustmentStatusRejected.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeAdjustments(t *testing.T) {
	items := []*Adjustment{sampleAdjustment(1), nil, sampleAdjustment(2), sampleAdjustment(3)}
	items[2].Status = AdjustmentStatusApplied
	sum := SummarizeAdjustments(items)
	if sum.Count != 3 || sum.ByStatus[AdjustmentStatusPending] != 2 || sum.ByStatus[AdjustmentStatusApplied] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalDelta != 1434 {
		t.Errorf("TotalDelta = %v, want 1434", sum.TotalDelta)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopAdjustmentsByDelta(items[2:], 1); len(top) != 1 || top[0].ID != "adjustment-003" {
		t.Errorf("top = %v", top)
	}
}
