package fraud

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleRule(n int) *Rule {
	return &Rule{
		ID:         fmt.Sprintf("rule-%03d", n),
		Name:       fmt.Sprintf("name %d;a=b,c\\d", n),
		Weight:     n*14 + 133,
		Enabled:    n%2 == 0,
		Expression: fmt.Sprintf("expression-%d", n),
		Status:     RuleStatusDraft,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestRuleCodecRoundTrip(t *testing.T) {
	in := sampleRule(4)
	line := EncodeRule(in)
	out, err := DecodeRule(line)
	if err != nil {
		t.Fatalf("DecodeRule(%q): %v", line, err)
	}
	if got := EncodeRule(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Name != in.Name {
		t.Errorf("Name = %q, want %q", out.Name, in.Name)
	}
	if out.Expression != in.Expression {
		t.Errorf("Expression = %q, want %q", out.Expression, in.Expression)
	}
	if _, err := DecodeRule("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryRuleStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryRuleStore()
	for n := 1; n <= 3; n++ {
		if err := st.Save(ctx, sampleRule(n)); err != nil {
			t.Fatalf("Save(%d): %v", n, err)
		}
	}
	got, err := st.Find(ctx, "rule-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = RuleStatusRetired
	again, _ := st.Find(ctx, "rule-002")
	if again.Status != RuleStatusDraft {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Save(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.All(ctx, RuleFilter{Status: RuleStatusDraft})
	if err != nil || len(list) != 2 || list[0].ID != "rule-001" {
		t.Fatalf("All by status = %v, %v", list, err)
	}
	if err := st.Remove(ctx, "rule-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Find(ctx, "rule-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestRuleStatusTransitions(t *testing.T) {
	for _, s := range []RuleStatus{RuleStatusDraft, RuleStatusRetired} {
		got, err := ParseRuleStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !RuleStatusDraft.CanTransition(RuleStatusLive) {
		t.Fatal("first transition should be allowed")
	}
	if RuleStatusLive.CanTransition(RuleStatusDraft) {
		t.Fatal("backwards transition allowed")
	}
	if !RuleStatusRetired.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeRules(t *testing.T) {
	items := []*Rule{sampleRule(1), nil, sampleRule(2), sampleRule(3)}
	items[2].Status = RuleStatusLive
	sum := SummarizeRules(items)
	if sum.Count != 3 || sum.ByStatus[RuleStatusDraft] != 2 || sum.ByStatus[RuleStatusLive] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalWeight != 483 {
		t.Errorf("TotalWeight = %v, want 483", sum.TotalWeight)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopRulesByWeight(items[2:], 1); len(top) != 1 || top[0].ID != "rule-003" {
		t.Errorf("top = %v", top)
	}
}
