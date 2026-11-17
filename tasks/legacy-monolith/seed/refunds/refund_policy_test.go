package refunds

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleRefundPolicy(n int) *RefundPolicy {
	return &RefundPolicy{
		ID:              fmt.Sprintf("refund_policy-%03d", n),
		Name:            fmt.Sprintf("name %d;a=b,c\\d", n),
		WindowDays:      n*8 + 164,
		RestockFeePct:   n*12 + 430,
		RequiresReceipt: n%2 == 0,
		Status:          RefundPolicyStatusActive,
		CreatedAt:       time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestRefundPolicyCodecRoundTrip(t *testing.T) {
	in := sampleRefundPolicy(4)
	line := EncodeRefundPolicy(in)
	out, err := DecodeRefundPolicy(line)
	if err != nil {
		t.Fatalf("DecodeRefundPolicy(%q): %v", line, err)
	}
	if got := EncodeRefundPolicy(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Name != in.Name {
		t.Errorf("Name = %q, want %q", out.Name, in.Name)
	}
	if _, err := DecodeRefundPolicy("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryRefundPolicyStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryRefundPolicyStore()
	for n := 1; n <= 3; n++ {
		if err := st.Put(ctx, sampleRefundPolicy(n)); err != nil {
			t.Fatalf("Put(%d): %v", n, err)
		}
	}
	got, err := st.Get(ctx, "refund_policy-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = RefundPolicyStatusArchived
	again, _ := st.Get(ctx, "refund_policy-002")
	if again.Status != RefundPolicyStatusActive {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Put(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.List(ctx, RefundPolicyFilter{Status: RefundPolicyStatusActive})
	if err != nil || len(list) != 2 || list[0].ID != "refund_policy-001" {
		t.Fatalf("List by status = %v, %v", list, err)
	}
	if err := st.Delete(ctx, "refund_policy-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, "refund_policy-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestRefundPolicyValidate(t *testing.T) {
	if err := sampleRefundPolicy(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleRefundPolicy(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleRefundPolicy(1)
	bad.Name = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing name: err = %v", err)
	}
	bad = sampleRefundPolicy(1)
	bad.Status = RefundPolicyStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestRefundPolicyStatusTransitions(t *testing.T) {
	for _, s := range []RefundPolicyStatus{RefundPolicyStatusActive, RefundPolicyStatusArchived} {
		got, err := ParseRefundPolicyStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !RefundPolicyStatusActive.CanTransition(RefundPolicyStatusArchived) {
		t.Fatal("first transition should be allowed")
	}
	if RefundPolicyStatusArchived.CanTransition(RefundPolicyStatusActive) {
		t.Fatal("backwards transition allowed")
	}
	if !RefundPolicyStatusArchived.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
