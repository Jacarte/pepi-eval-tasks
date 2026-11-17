package pricing

import (
	"context"
	"testing"
)

func TestEngineQuote(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryPriceRuleStore()
	for _, r := range []*PriceRule{
		{ID: "r-all", SKU: "*", Kind: KindPercent, Value: 1000, Priority: 1, Stackable: true, Status: PriceRuleStatusActive},
		{ID: "r-fixed", SKU: "SKU-1", Kind: KindFixed, Value: 50, Priority: 2, Status: PriceRuleStatusActive},
		{ID: "r-late", SKU: "SKU-1", Kind: KindFixed, Value: 999, Priority: 3, Status: PriceRuleStatusActive},
		{ID: "r-draft", SKU: "SKU-1", Kind: KindFixed, Value: 999, Priority: 0, Status: PriceRuleStatusDraft},
	} {
		if err := store.Put(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	q, err := NewEngine(store).Quote(ctx, "SKU-1", 1000, 3)
	if err != nil {
		t.Fatal(err)
	}
	if q.Discount != 150 || q.Unit != 850 || q.Total != 2550 || len(q.Applied) != 2 {
		t.Fatalf("quote = %+v", q)
	}
}

func TestApplyCoupon(t *testing.T) {
	if ApplyCoupon(1000, 15) != 850 || ApplyCoupon(1000, 0) != 1000 || ApplyCoupon(1000, 120) != 0 {
		t.Fatal("ApplyCoupon")
	}
}
