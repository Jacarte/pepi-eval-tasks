package orders

import (
	"context"
	"fmt"
	"strings"

	"shopcore/money"
	"shopcore/pricing"
	"shopcore/tax"
)

// PlaceRequest is the input to Place.
type PlaceRequest struct {
	CustomerID string
	Region     string // tax region, e.g. "US-CA"
	Lines      []Line
	CouponCode string
	PercentOff int // resolved by the caller from CouponCode
}

// Place validates and prices the request and stores a new order.
func (s *Service) Place(ctx context.Context, req PlaceRequest) (*Order, error) {
	if strings.TrimSpace(req.CustomerID) == "" {
		return nil, ErrMissingCustomer
	}
	if len(req.Lines) == 0 {
		return nil, ErrEmptyOrder
	}
	if len(req.Lines) > MaxLines {
		return nil, ErrTooManyLines
	}
	for _, l := range req.Lines {
		if l.Qty <= 0 {
			return nil, fmt.Errorf("line %s: %w", l.SKU, ErrInvalidQuantity)
		}
	}
	now := s.clock.Now()
	o := &Order{
		ID:         s.ids.Next("ord"),
		CustomerID: req.CustomerID,
		Region:     strings.ToUpper(req.Region),
		Lines:      append([]Line(nil), req.Lines...),
		CouponCode: req.CouponCode,
		Status:     StatusPlaced,
		PlacedAt:   now,
		UpdatedAt:  now,
	}
	for _, l := range o.Lines {
		o.Subtotal += l.Total()
	}
	if req.PercentOff > 0 {
		o.Discount = o.Subtotal - pricing.ApplyCoupon(o.Subtotal, req.PercentOff)
	}
	t, err := tax.Compute(o.Region, o.Subtotal-o.Discount)
	if err != nil {
		return nil, err
	}
	o.Tax = t
	o.Total = money.Sum(o.Subtotal, -o.Discount, o.Tax)
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}
	return o, nil
}
