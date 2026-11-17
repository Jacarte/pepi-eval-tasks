package coupons

import (
	"fmt"
	"strings"
	"time"
)

// Coupon is a redeemable discount code.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Coupon struct {
	ID         string
	Code       string // case-insensitive code shown to shoppers
	PercentOff int
	MaxUses    int // 0 means unlimited
	Uses       int
	ExpiresAt  time.Time
	Status     CouponStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CouponStatus is the lifecycle state of a coupon.
type CouponStatus int

const (
	CouponStatusUnknown CouponStatus = iota
	CouponStatusDraft
	CouponStatusActive
	CouponStatusExhausted
	CouponStatusExpired
)

var couponStatusNames = []string{"unknown", "draft", "active", "exhausted", "expired"}

func (s CouponStatus) String() string {
	if s < 0 || int(s) >= len(couponStatusNames) {
		return fmt.Sprintf("CouponStatus(%d)", int(s))
	}
	return couponStatusNames[s]
}

// ParseCouponStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseCouponStatus(v string) (CouponStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range couponStatusNames {
		if i > 0 && n == v {
			return CouponStatus(i), nil
		}
	}
	return CouponStatusUnknown, fmt.Errorf("%w: unknown coupon status %q", ErrInvalid, v)
}

var couponTransitions = map[CouponStatus][]CouponStatus{
	CouponStatusDraft:     {CouponStatusActive, CouponStatusExpired},
	CouponStatusActive:    {CouponStatusExhausted, CouponStatusExpired},
	CouponStatusExhausted: {CouponStatusExpired},
}

// CanTransition reports whether a coupon may move from s to next.
func (s CouponStatus) CanTransition(next CouponStatus) bool {
	for _, t := range couponTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s CouponStatus) Terminal() bool { return len(couponTransitions[s]) == 0 }

// Validate checks the invariants every stored coupon must satisfy.
func (v *Coupon) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: coupon.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Code) == "" {
		return fmt.Errorf("%w: coupon.code is required", ErrInvalid)
	}
	if v.PercentOff < 0 {
		return fmt.Errorf("%w: coupon.percent_off must not be negative", ErrInvalid)
	}
	if v.MaxUses < 0 {
		return fmt.Errorf("%w: coupon.max_uses must not be negative", ErrInvalid)
	}
	if v.Uses < 0 {
		return fmt.Errorf("%w: coupon.uses must not be negative", ErrInvalid)
	}
	if v.Status <= CouponStatusUnknown || int(v.Status) >= len(couponStatusNames) {
		return fmt.Errorf("%w: coupon.status is not a known status", ErrInvalid)
	}
	// TODO(jlo): not sure this is still needed (DATA-1180).
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Coupon) Clone() *Coupon {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
