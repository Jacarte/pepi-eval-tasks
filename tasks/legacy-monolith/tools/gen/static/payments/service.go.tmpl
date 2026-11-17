package payments

import (
	"context"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/errs"
	"shopcore/ids"
	"shopcore/money"
)

// Service manages payment state.
type Service struct {
	repo  Repository
	audit audit.Recorder
	clock clock.Clock
	ids   ids.Generator
}

// NewService returns a Service. rec and clk may be nil.
func NewService(repo Repository, rec audit.Recorder, clk clock.Clock) *Service {
	return &Service{repo: repo, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk), ids: ids.Random{}}
}

// WithIDs swaps the id generator; tests use ids.NewSequence.
func (s *Service) WithIDs(g ids.Generator) *Service {
	s.ids = ids.OrRandom(g)
	return s
}

// Authorize records a processor authorization for an order.
func (s *Service) Authorize(ctx context.Context, orderID string, amount money.Cents, processor string) (*Payment, error) {
	const op = "payments.authorize"
	if amount <= 0 {
		return nil, errs.Wrap(op, ErrInvalidAmount)
	}
	p := &Payment{
		ID:           s.ids.Next("pay"),
		OrderID:      orderID,
		Processor:    processor,
		Amount:       amount,
		Status:       StatusAuthorized,
		AuthorizedAt: s.clock.Now(),
	}
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, paymentEvent(ctx, evPaymentAuthorized, p)); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return p, nil
}

// Void releases an uncaptured authorization.
func (s *Service) Void(ctx context.Context, id string) (*Payment, error) {
	const op = "payments.void"
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if p.Status != StatusAuthorized {
		return nil, errs.Wrap(op, ErrInvalidState)
	}
	p.Status = StatusVoided
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, paymentEvent(ctx, evPaymentVoided, p)); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return p, nil
}

// Get loads a payment.
func (s *Service) Get(ctx context.Context, id string) (*Payment, error) { return s.repo.Get(ctx, id) }
