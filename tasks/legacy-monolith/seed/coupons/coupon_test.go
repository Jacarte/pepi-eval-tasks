package coupons

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleCoupon(n int) *Coupon {
	return &Coupon{
		ID:         fmt.Sprintf("coupon-%03d", n),
		Code:       fmt.Sprintf("code %d;a=b,c\\d", n),
		PercentOff: n*27 + 181,
		MaxUses:    n*15 + 189,
		Uses:       n*4 + 165,
		ExpiresAt:  time.Date(2023, time.March, n, 9, 30, 0, 0, time.UTC),
		Status:     CouponStatusDraft,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestCouponCodecRoundTrip(t *testing.T) {
	in := sampleCoupon(4)
	line := EncodeCoupon(in)
	out, err := DecodeCoupon(line)
	if err != nil {
		t.Fatalf("DecodeCoupon(%q): %v", line, err)
	}
	if got := EncodeCoupon(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Code != in.Code {
		t.Errorf("Code = %q, want %q", out.Code, in.Code)
	}
	if _, err := DecodeCoupon("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryCouponStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryCouponStore()
	for n := 1; n <= 3; n++ {
		if err := st.Store(ctx, sampleCoupon(n)); err != nil {
			t.Fatalf("Store(%d): %v", n, err)
		}
	}
	got, err := st.Load(ctx, "coupon-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = CouponStatusExpired
	again, _ := st.Load(ctx, "coupon-002")
	if again.Status != CouponStatusDraft {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Store(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.Scan(ctx, CouponFilter{Status: CouponStatusDraft})
	if err != nil || len(list) != 2 || list[0].ID != "coupon-001" {
		t.Fatalf("Scan by status = %v, %v", list, err)
	}
	if err := st.Drop(ctx, "coupon-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Load(ctx, "coupon-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}
