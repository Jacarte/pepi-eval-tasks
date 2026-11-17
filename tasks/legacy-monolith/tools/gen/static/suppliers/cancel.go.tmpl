package suppliers

import (
	"context"
	"errors"
	"fmt"
)

// ErrNotCancellable is returned when a purchase order has already been
// received or cancelled.
var ErrNotCancellable = errors.New("suppliers: purchase order can no longer be cancelled")

// Cancel withdraws a draft or approved purchase order.
func (s *Service) Cancel(ctx context.Context, poID string) (*PurchaseOrder, error) {
	po, err := s.pos.Find(ctx, poID)
	if err != nil {
		return nil, err
	}
	if po.Status != PurchaseOrderStatusDraft && po.Status != PurchaseOrderStatusApproved {
		return nil, ErrNotCancellable
	}
	po.Status = PurchaseOrderStatusCancelled
	po.UpdatedAt = s.clock.Now()
	if err := s.pos.Save(ctx, po); err != nil {
		return nil, fmt.Errorf("cancel %s: %w", po.ID, err)
	}
	return po, nil
}
