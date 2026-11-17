package fraud

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleReview(n int) *Review {
	return &Review{
		ID:        fmt.Sprintf("review-%03d", n),
		OrderID:   fmt.Sprintf("orderid %d;a=b,c\\d", n),
		Score:     n*25 + 332,
		Reason:    fmt.Sprintf("reason %d;a=b,c\\d", n),
		Analyst:   fmt.Sprintf("analyst-%d", n),
		Status:    ReviewStatusOpen,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestReviewCodecRoundTrip(t *testing.T) {
	in := sampleReview(4)
	line := EncodeReview(in)
	out, err := DecodeReview(line)
	if err != nil {
		t.Fatalf("DecodeReview(%q): %v", line, err)
	}
	if got := EncodeReview(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.OrderID != in.OrderID {
		t.Errorf("OrderID = %q, want %q", out.OrderID, in.OrderID)
	}
	if out.Reason != in.Reason {
		t.Errorf("Reason = %q, want %q", out.Reason, in.Reason)
	}
	if out.Analyst != in.Analyst {
		t.Errorf("Analyst = %q, want %q", out.Analyst, in.Analyst)
	}
	if _, err := DecodeReview("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryReviewStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryReviewStore()
	for n := 1; n <= 3; n++ {
		if err := st.Save(ctx, sampleReview(n)); err != nil {
			t.Fatalf("Save(%d): %v", n, err)
		}
	}
	got, err := st.Find(ctx, "review-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = ReviewStatusConfirmed
	again, _ := st.Find(ctx, "review-002")
	if again.Status != ReviewStatusOpen {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Save(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.All(ctx, ReviewFilter{Status: ReviewStatusOpen})
	if err != nil || len(list) != 2 || list[0].ID != "review-001" {
		t.Fatalf("All by status = %v, %v", list, err)
	}
	if err := st.Remove(ctx, "review-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Find(ctx, "review-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestReviewValidate(t *testing.T) {
	if err := sampleReview(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleReview(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleReview(1)
	bad.OrderID = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing order id: err = %v", err)
	}
	bad = sampleReview(1)
	bad.Status = ReviewStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestReviewStatusTransitions(t *testing.T) {
	for _, s := range []ReviewStatus{ReviewStatusOpen, ReviewStatusConfirmed} {
		got, err := ParseReviewStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ReviewStatusOpen.CanTransition(ReviewStatusEscalated) {
		t.Fatal("first transition should be allowed")
	}
	if ReviewStatusEscalated.CanTransition(ReviewStatusOpen) {
		t.Fatal("backwards transition allowed")
	}
	if !ReviewStatusConfirmed.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
