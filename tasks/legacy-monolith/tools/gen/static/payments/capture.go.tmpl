package payments

import (
	"context"

	"shopcore/money"
)

// Capture captures amount of an authorized payment. Partial captures are
// allowed; the remainder of the authorization is released by the processor.
func (s *Service) Capture(ctx context.Context, id string, amount money.Cents) (*Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != StatusAuthorized {
		return nil, ErrInvalidState
	}
	if amount > p.Amount {
		return nil, ErrExceedsAuthorization
	}
	p.Captured = amount
	p.Status = StatusCaptured
	p.CapturedAt = s.clock.Now()
	if err := s.repo.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
