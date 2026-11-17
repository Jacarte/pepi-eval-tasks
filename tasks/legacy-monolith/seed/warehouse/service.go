// Package warehouse manages physical bin locations and internal transfers.
package warehouse

import (
	"context"
	"errors"
	"time"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

var (
	ErrBinNotFree   = errors.New("warehouse: bin is not free")
	ErrBinBlocked   = errors.New("warehouse: bin is blocked")
	ErrSameBin      = errors.New("warehouse: transfer source and destination are the same")
	ErrMissingSKU   = errors.New("warehouse: sku is required")
	ErrOverCapacity = errors.New("warehouse: quantity exceeds bin capacity")
)

// Service owns bin allocation.
type Service struct {
	bins      BinSlotStore
	transfers TransferStore
	rec       audit.Recorder
	now       func() time.Time // predates the clock package; kept as a func
	ids       ids.Generator
}

func NewService(bins BinSlotStore, transfers TransferStore, rec audit.Recorder, clk clock.Clock, gen ids.Generator) *Service {
	c := clock.OrSystem(clk)
	return &Service{bins: bins, transfers: transfers, rec: audit.OrDiscard(rec), now: c.Now, ids: ids.OrRandom(gen)}
}

// Allocate assigns a free bin to a SKU.
func (s *Service) Allocate(ctx context.Context, binID, sku string) (*BinSlot, error) {
	const op = "warehouse.allocate"
	if sku == "" {
		return nil, errs.Wrap(op, ErrMissingSKU)
	}
	bin, err := s.bins.Load(ctx, binID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if bin.Status != BinSlotStatusFree {
		return nil, errs.Wrap(op, ErrBinNotFree)
	}
	bin.SKU = sku
	bin.Status = BinSlotStatusAllocated
	bin.UpdatedAt = s.now()
	if err = s.bins.Store(ctx, bin); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err = s.rec.Record(ctx, audit.Event{Name: "bin_slot.allocated", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: bin.ID, Attrs: map[string]string{"sku": sku}}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return bin, nil
}

// Block takes a bin out of service (damage, pest control, ...).
func (s *Service) Block(ctx context.Context, binID string) (*BinSlot, error) {
	const op = "warehouse.block"
	bin, err := s.bins.Load(ctx, binID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	bin.Status = BinSlotStatusBlocked
	bin.UpdatedAt = s.now()
	if err = s.bins.Store(ctx, bin); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err = s.rec.Record(ctx, audit.Event{Name: "bin_slot.blocked", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: bin.ID}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return bin, nil
}

// PlanTransfer schedules a move between bins. Transfers are internal and are
// not audited; the WMS keeps its own movement log.
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
