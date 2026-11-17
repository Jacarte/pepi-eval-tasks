package orders

import (
	"context"
	"strconv"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/errs"
	"shopcore/ids"
)

// Config wires a Service. Repo is required; everything else has a default.
type Config struct {
	Repo  Repository
	Audit audit.Recorder
	Clock clock.Clock
	IDs   ids.Generator
}

// Service is the orders application service.
type Service struct {
	repo  Repository
	audit audit.Recorder
	clock clock.Clock
	ids   ids.Generator
}

func NewService(cfg Config) *Service {
	return &Service{
		repo:  cfg.Repo,
		audit: audit.OrDiscard(cfg.Audit),
		clock: clock.OrSystem(cfg.Clock),
		ids:   ids.OrRandom(cfg.IDs),
	}
}

// Get loads one order.
func (s *Service) Get(ctx context.Context, id string) (*Order, error) {
	return s.repo.Get(ctx, id)
}

// ForCustomer lists a customer's orders, oldest id first.
func (s *Service) ForCustomer(ctx context.Context, customerID string) ([]*Order, error) {
	return s.repo.ListByCustomer(ctx, customerID)
}

// Cancel cancels a placed order.
func (s *Service) Cancel(ctx context.Context, id string) (*Order, error) {
	const op = "orders.cancel"
	o, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if o.Status != StatusPlaced {
		return nil, errs.Wrap(op, ErrNotCancellable)
	}
	o.Status = StatusCancelled
	o.UpdatedAt = s.clock.Now()
	if err := s.repo.Save(ctx, o); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.emit(ctx, evOrderCancelled, o, nil); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return o, nil
}

// MarkFulfilled is called by the warehouse integration once every line has
// shipped.
func (s *Service) MarkFulfilled(ctx context.Context, id string) error {
	const op = "orders.markfulfilled"
	o, err := s.repo.Get(ctx, id)
	if err != nil {
		return errs.Wrap(op, err)
	}
	o.Status = StatusFulfilled
	o.UpdatedAt = s.clock.Now()
	if err := s.repo.Save(ctx, o); err != nil {
		return errs.Wrap(op, err)
	}
	return errs.Wrap(op, s.emit(ctx, evOrderFulfilled, o, map[string]string{"items": strconv.Itoa(o.ItemCount())}))
}
