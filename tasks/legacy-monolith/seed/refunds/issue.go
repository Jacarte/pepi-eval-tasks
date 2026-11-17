package refunds

import (
	"context"
	"fmt"
	"strings"

	"shopcore/money"
)

// IssueRequest is the input to Issue.
type IssueRequest struct {
	PaymentID string
	OrderID   string
	Amount    money.Cents
	Reason    Reason
	Note      string
}

// Issue creates a refund in the issued state.
func (s *Service) Issue(ctx context.Context, req IssueRequest) (*Refund, error) {
	if strings.TrimSpace(req.PaymentID) == "" {
		return nil, fmt.Errorf("refunds: %w", ErrMissingPayment)
	}
	if req.Amount <= 0 {
		return nil, fmt.Errorf("refunds: %w", ErrInvalidAmount)
	}
	if !knownReasons[req.Reason] {
		return nil, fmt.Errorf("refunds: %w: %q", ErrUnknownReason, req.Reason)
	}
	if s.max > 0 && req.Amount > s.max {
		return nil, fmt.Errorf("refunds: %w", ErrOverLimit)
	}
	r := &Refund{
		ID:        s.ids.Next("rfd"),
		PaymentID: req.PaymentID,
		OrderID:   req.OrderID,
		Amount:    req.Amount,
		Reason:    req.Reason,
		Note:      req.Note,
		Status:    StatusIssued,
		IssuedAt:  s.clock.Now(),
	}
	if err := s.repo.Insert(ctx, r); err != nil {
		return nil, fmt.Errorf("refunds: insert: %w", err)
	}
	return r, nil
}
