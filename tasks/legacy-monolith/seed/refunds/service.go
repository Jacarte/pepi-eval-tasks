package refunds

import (
	"context"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/errs"
	"shopcore/ids"
	"shopcore/money"
)

// Deps are the collaborators of a Service. Repo is required.
type Deps struct {
	Repo  Repository
	Audit audit.Recorder
	Clock clock.Clock
	IDs   ids.Generator
	// MaxAmount caps a single refund; zero means no cap. Support leads can
	// issue above the cap through the back office, which bypasses Issue.
	MaxAmount money.Cents
}

type Service struct {
	repo  Repository
	audit audit.Recorder
	clock clock.Clock
	ids   ids.Generator
	max   money.Cents
}

// New builds a Service from d.
func New(d Deps) *Service {
	return &Service{
		repo:  d.Repo,
		audit: audit.OrDiscard(d.Audit),
		clock: clock.OrSystem(d.Clock),
		ids:   ids.OrRandom(d.IDs),
		max:   d.MaxAmount,
	}
}

// Settle marks a refund as paid out by the processor.
func (s *Service) Settle(ctx context.Context, id string) (*Refund, error) {
	const op = "refunds.settle"
	r, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if r.Status != StatusIssued {
		return nil, errs.Wrap(op, ErrNotIssued)
	}
	r.Status = StatusSettled
	r.SettledAt = s.clock.Now()
	if err := s.repo.Update(ctx, r); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.record(ctx, evRefundSettled, r); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return r, nil
}

// TotalForPayment sums non-rejected refunds against a payment.
func (s *Service) TotalForPayment(ctx context.Context, paymentID string) (money.Cents, error) {
	list, err := s.repo.ForPayment(ctx, paymentID)
	if err != nil {
		return 0, err
	}
	var total money.Cents
	for _, r := range list {
		if r.Status != StatusRejected {
			total += r.Amount
		}
	}
	return total, nil
}
