package loyalty

import (
	"context"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/errs"
)

// Service runs the points ledger for loyalty accounts.
type Service struct {
	repo  Repository
	audit audit.Recorder
	clock clock.Clock
}

func NewService(repo Repository, rec audit.Recorder, clk clock.Clock) *Service {
	return &Service{repo: repo, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk)}
}

// Open creates an empty account for a customer. The account id is derived
// from the customer id so re-opening is idempotent.
func (s *Service) Open(ctx context.Context, customerID string) (*Account, error) {
	id := "loy-" + customerID
	if a, err := s.repo.Load(ctx, id); err == nil {
		return a, nil
	}
	a := &Account{ID: id, CustomerID: customerID, TierName: tierFor(0), UpdatedAt: s.clock.Now()}
	if err := s.repo.Store(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// Accrue adds points, e.g. after an order is fulfilled.
func (s *Service) Accrue(ctx context.Context, accountID string, points int64) (*Account, error) {
	const op = "loyalty.accrue"
	if points <= 0 {
		return nil, errs.Wrap(op, ErrInvalidPoints)
	}
	a, err := s.repo.Load(ctx, accountID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	a.Balance += points
	a.Lifetime += points
	a.TierName = tierFor(a.Lifetime)
	a.UpdatedAt = s.clock.Now()
	if err := s.repo.Store(ctx, a); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, s.event(ctx, evAccountAccrued, a, points)); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return a, nil
}
