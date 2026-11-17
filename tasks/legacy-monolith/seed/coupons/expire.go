package coupons

import (
	"context"
	"fmt"
)

// Expire ends a coupon before its expiry date, e.g. when a code leaks.
func (s *Service) Expire(ctx context.Context, id string) (*Coupon, error) {
	c, err := s.coupons.Load(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("coupons: %w", err)
	}
	if c.Status != CouponStatusActive && c.Status != CouponStatusDraft {
		return nil, fmt.Errorf("coupons: %w", ErrNotActive)
	}
	c.Status = CouponStatusExpired
	c.ExpiresAt = s.clock.Now()
	c.UpdatedAt = c.ExpiresAt
	if err := s.coupons.Store(ctx, c); err != nil {
		return nil, fmt.Errorf("coupons: %w", err)
	}
	return c, nil
}
