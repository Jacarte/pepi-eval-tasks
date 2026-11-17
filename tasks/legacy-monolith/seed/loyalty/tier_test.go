package loyalty

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleTier(n int) *Tier {
	return &Tier{
		ID:         fmt.Sprintf("tier-%03d", n),
		Name:       fmt.Sprintf("name %d;a=b,c\\d", n),
		Threshold:  int64(n*22 + 203),
		Multiplier: n*22 + 339,
		Status:     TierStatusActive,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestTierCodecRoundTrip(t *testing.T) {
	in := sampleTier(4)
	line := EncodeTier(in)
	out, err := DecodeTier(line)
	if err != nil {
		t.Fatalf("DecodeTier(%q): %v", line, err)
	}
	if got := EncodeTier(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Name != in.Name {
		t.Errorf("Name = %q, want %q", out.Name, in.Name)
	}
	if _, err := DecodeTier("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryTierStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryTierStore()
	for n := 1; n <= 3; n++ {
		if err := st.Save(ctx, sampleTier(n)); err != nil {
			t.Fatalf("Save(%d): %v", n, err)
		}
	}
	got, err := st.Find(ctx, "tier-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = TierStatusRetired
	again, _ := st.Find(ctx, "tier-002")
	if again.Status != TierStatusActive {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Save(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.All(ctx, TierFilter{Status: TierStatusActive})
	if err != nil || len(list) != 2 || list[0].ID != "tier-001" {
		t.Fatalf("All by status = %v, %v", list, err)
	}
	if err := st.Remove(ctx, "tier-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Find(ctx, "tier-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestSummarizeTiers(t *testing.T) {
	items := []*Tier{sampleTier(1), nil, sampleTier(2), sampleTier(3)}
	items[2].Status = TierStatusRetired
	sum := SummarizeTiers(items)
	if sum.Count != 3 || sum.ByStatus[TierStatusActive] != 2 || sum.ByStatus[TierStatusRetired] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalThreshold != 741 {
		t.Errorf("TotalThreshold = %v, want 741", sum.TotalThreshold)
	}
	if sum.TotalMultiplier != 1149 {
		t.Errorf("TotalMultiplier = %v, want 1149", sum.TotalMultiplier)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopTiersByThreshold(items[2:], 1); len(top) != 1 || top[0].ID != "tier-003" {
		t.Errorf("top = %v", top)
	}
}
