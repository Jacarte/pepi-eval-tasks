// Package coupons manages discount codes.
package coupons

import (
	"context"
	"errors"
	"strings"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

var (
	ErrNotDraft  = errors.New("coupons: coupon is not a draft")
	ErrNotActive = errors.New("coupons: coupon is not active")
	ErrExpired   = errors.New("coupons: coupon has expired")
)

type Service struct {
	coupons CouponStore
	audit   audit.Recorder
	clock   clock.Clock
}

func NewService(store CouponStore, rec audit.Recorder, clk clock.Clock) *Service {
	return &Service{coupons: store, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk)}
}

// Activate makes a draft coupon redeemable.
func (s *Service) Activate(ctx context.Context, id string) (*Coupon, error) {
	c, err := s.coupons.Load(ctx, id)
	if err != nil {
		return nil, errs.Wrap("coupons.activate", err)
	}
	if c.Status != CouponStatusDraft {
		return nil, errs.Wrap("coupons.activate", ErrNotDraft)
	}
	c.Status = CouponStatusActive
	c.Code = strings.ToUpper(c.Code)
	c.UpdatedAt = s.clock.Now()
	if err := s.coupons.Store(ctx, c); err != nil {
		return nil, errs.Wrap("coupons.activate", err)
	}
	if err := s.audit.Record(ctx, audit.Event{Name: "coupon.activated", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: c.ID}); err != nil {
		return nil, errs.Wrap("coupons.activate", err)
	}
	return c, nil
}

// Redeem consumes one use of a coupon and returns its percentage.
func (s *Service) Redeem(ctx context.Context, id string) (*Coupon, error) {
	c, err := s.coupons.Load(ctx, id)
	if err != nil {
		return nil, errs.Wrap("coupons.redeem", err)
	}
	if c.Status != CouponStatusActive {
		return nil, errs.Wrap("coupons.redeem", ErrNotActive)
	}
	now := s.clock.Now()
	if !c.ExpiresAt.IsZero() && !now.Before(c.ExpiresAt) {
		return nil, errs.Wrap("coupons.redeem", ErrExpired)
	}
	c.Uses++
	if c.MaxUses > 0 && c.Uses >= c.MaxUses {
		c.Status = CouponStatusExhausted
	}
	c.UpdatedAt = now
	if err := s.coupons.Store(ctx, c); err != nil {
		return nil, errs.Wrap("coupons.redeem", err)
	}
	if err := s.audit.Record(ctx, audit.Event{Name: "coupon.redeemed", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: c.ID}); err != nil {
		return nil, errs.Wrap("coupons.redeem", err)
	}
	return c, nil
}
