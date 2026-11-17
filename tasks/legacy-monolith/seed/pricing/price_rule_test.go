package pricing

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func samplePriceRule(n int) *PriceRule {
	return &PriceRule{
		ID:        fmt.Sprintf("price_rule-%03d", n),
		SKU:       fmt.Sprintf("sku %d;a=b,c\\d", n),
		Kind:      fmt.Sprintf("kind-%d", n),
		Value:     int64(n*26 + 9),
		Priority:  n*23 + 471,
		Stackable: n%2 == 0,
		Status:    PriceRuleStatusDraft,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestPriceRuleCodecRoundTrip(t *testing.T) {
	in := samplePriceRule(4)
	line := EncodePriceRule(in)
	out, err := DecodePriceRule(line)
	if err != nil {
		t.Fatalf("DecodePriceRule(%q): %v", line, err)
	}
	if got := EncodePriceRule(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.SKU != in.SKU {
		t.Errorf("SKU = %q, want %q", out.SKU, in.SKU)
	}
	if out.Kind != in.Kind {
		t.Errorf("Kind = %q, want %q", out.Kind, in.Kind)
	}
	if _, err := DecodePriceRule("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryPriceRuleStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryPriceRuleStore()
	for n := 1; n <= 3; n++ {
		if err := st.Put(ctx, samplePriceRule(n)); err != nil {
			t.Fatalf("Put(%d): %v", n, err)
		}
	}
	got, err := st.Get(ctx, "price_rule-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = PriceRuleStatusExpired
	again, _ := st.Get(ctx, "price_rule-002")
	if again.Status != PriceRuleStatusDraft {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Put(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.List(ctx, PriceRuleFilter{Status: PriceRuleStatusDraft})
	if err != nil || len(list) != 2 || list[0].ID != "price_rule-001" {
		t.Fatalf("List by status = %v, %v", list, err)
	}
	if err := st.Delete(ctx, "price_rule-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, "price_rule-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestPriceRuleValidate(t *testing.T) {
	if err := samplePriceRule(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := samplePriceRule(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = samplePriceRule(1)
	bad.SKU = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing sku: err = %v", err)
	}
	bad = samplePriceRule(1)
	bad.Status = PriceRuleStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestSummarizePriceRules(t *testing.T) {
	items := []*PriceRule{samplePriceRule(1), nil, samplePriceRule(2), samplePriceRule(3)}
	items[2].Status = PriceRuleStatusActive
	sum := SummarizePriceRules(items)
	if sum.Count != 3 || sum.ByStatus[PriceRuleStatusDraft] != 2 || sum.ByStatus[PriceRuleStatusActive] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalValue != 183 {
		t.Errorf("TotalValue = %v, want 183", sum.TotalValue)
	}
	if sum.TotalPriority != 1551 {
		t.Errorf("TotalPriority = %v, want 1551", sum.TotalPriority)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopPriceRulesByValue(items[2:], 1); len(top) != 1 || top[0].ID != "price_rule-003" {
		t.Errorf("top = %v", top)
	}
}
