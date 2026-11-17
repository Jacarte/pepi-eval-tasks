package fraud

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

func TestScore(t *testing.T) {
	rules := []*Rule{
		{ID: "r1", Name: "big", Weight: 300, Enabled: true, Expression: "amount > 100000", Status: RuleStatusLive},
		{ID: "r2", Name: "fresh", Weight: 200, Enabled: true, Expression: "age<2", Status: RuleStatusLive},
		{ID: "r3", Name: "off", Weight: 900, Enabled: false, Expression: "nomatch", Status: RuleStatusLive},
	}
	base := Score(Signals{Country: "US", AddressMatch: true, AccountAgeDays: 400}, rules)
	risky := Score(Signals{Country: "US", AmountCents: 200000, AccountAgeDays: 1}, rules)
	if risky-base != 500 {
		t.Fatalf("base=%d risky=%d", base, risky)
	}
	if Score(Signals{Country: "ZZ", AccountAgeDays: 99}, nil) != defaultCountryRisk {
		t.Fatal("unknown country should use the default")
	}
}

func TestDeskEscalateClear(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-f")
	store := NewMemoryReviewStore()
	rec := audit.NewMemory(nil)
	desk := NewDesk(store, rec, nil)
	_ = store.Save(ctx, &Review{ID: "rv-1", OrderID: "ord-1", Score: 700, Status: ReviewStatusOpen})
	if _, err := desk.Escalate(ctx, "rv-1", "kim"); err != nil {
		t.Fatal(err)
	}
	if _, err := desk.Escalate(ctx, "rv-1", "kim"); !errors.Is(err, ErrNotOpen) {
		t.Fatalf("err = %v", err)
	}
	if _, err := desk.Clear(ctx, "rv-1"); err != nil {
		t.Fatal(err)
	}
	if evs := rec.Events(); len(evs) != 2 || evs[0].Name != "review.escalated" || evs[1].Name != "review.cleared" {
		t.Fatalf("events = %v", evs)
	}
}
