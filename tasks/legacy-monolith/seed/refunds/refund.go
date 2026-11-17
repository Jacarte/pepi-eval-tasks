// Package refunds issues and settles customer refunds against payments.
package refunds

import (
	"errors"
	"time"

	"shopcore/money"
)

// Reason codes accepted by Issue. They map 1:1 to the support macro list.
type Reason string

const (
	ReasonDamaged     Reason = "damaged"
	ReasonNotReceived Reason = "not_received"
	ReasonChangedMind Reason = "changed_mind"
	ReasonDuplicate   Reason = "duplicate_charge"
	ReasonGoodwill    Reason = "goodwill"
)

var knownReasons = map[Reason]bool{
	ReasonDamaged: true, ReasonNotReceived: true, ReasonChangedMind: true, ReasonDuplicate: true, ReasonGoodwill: true,
}

// Status of a refund.
type Status string

const (
	StatusIssued   Status = "issued"
	StatusSettled  Status = "settled"
	StatusRejected Status = "rejected"
)

var (
	ErrInvalidAmount  = errors.New("refunds: amount must be positive")
	ErrUnknownReason  = errors.New("refunds: unknown reason")
	ErrMissingPayment = errors.New("refunds: payment id is required")
	ErrOverLimit      = errors.New("refunds: amount over the configured limit")
	ErrConflict       = errors.New("refunds: refund already exists")
	ErrNotIssued      = errors.New("refunds: refund is not in issued state")
)

// Refund is money returned to a customer.
type Refund struct {
	ID        string
	PaymentID string
	OrderID   string
	Amount    money.Cents
	Reason    Reason
	Note      string
	Status    Status
	IssuedAt  time.Time
	SettledAt time.Time
}

// Clone returns a copy.
func (r *Refund) Clone() *Refund {
	if r == nil {
		return nil
	}
	cp := *r
	return &cp
}
