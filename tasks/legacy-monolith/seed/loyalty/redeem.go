package loyalty

import (
	"context"

	"shopcore/errs"
)

// Redeem spends points from an account at checkout.
func (s *Service) Redeem(ctx context.Context, accountID string, points int64) (*Account, error) {
	if points <= 0 {
		return nil, errs.Wrap("Redeem", ErrInvalidPoints)
	}
	a, err := s.repo.Load(ctx, accountID)
	if err != nil {
		return nil, errs.Wrap("Redeem", err)
	}
	if a.Balance < points {
		return nil, errs.Wrap("Redeem", ErrInsufficientPoints)
	}
	a.Balance -= points
	a.UpdatedAt = s.clock.Now()
	if err := s.repo.Store(ctx, a); err != nil {
		return nil, errs.Wrap("Redeem", err)
	}
	return a, nil
}
