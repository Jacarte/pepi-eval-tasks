package returns

import (
	"context"

	"shopcore/errs"
)

// Request opens a return on behalf of the customer.
func (s *Service) Request(ctx context.Context, orderID, sku string, qty int, reason string) (*ReturnRequest, error) {
	const op = "returns.request"
	if qty <= 0 {
		return nil, errs.Wrap(op, ErrBadQuantity)
	}
	now := s.clock.Now()
	rr := &ReturnRequest{ID: s.ids.Next("rma"), OrderID: orderID, SKU: sku, Qty: qty, Reason: reason, Status: ReturnRequestStatusRequested, CreatedAt: now, UpdatedAt: now}
	if err := s.store.Save(ctx, rr); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return rr, nil
}
