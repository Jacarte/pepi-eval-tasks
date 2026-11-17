package warehouse

import (
	"context"

	"shopcore/errs"
)

// PlanTransfer schedules a move between bins.
func (s *Service) PlanTransfer(ctx context.Context, from, to, sku string, qty int) (*Transfer, error) {
	const op = "warehouse.plantransfer"
	if from == to {
		return nil, errs.Wrap(op, ErrSameBin)
	}
	dst, err := s.bins.Load(ctx, to)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if dst.Status == BinSlotStatusBlocked {
		return nil, errs.Wrap(op, ErrBinBlocked)
	}
	if qty > dst.Capacity {
		return nil, errs.Wrap(op, ErrOverCapacity)
	}
	now := s.now()
	tr := &Transfer{ID: s.ids.Next("trf"), FromBin: from, ToBin: to, SKU: sku, Qty: qty, Status: TransferStatusPlanned, CreatedAt: now, UpdatedAt: now}
	if err := s.transfers.Store(ctx, tr); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return tr, nil
}
