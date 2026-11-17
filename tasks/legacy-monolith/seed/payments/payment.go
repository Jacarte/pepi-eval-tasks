// Package payments tracks card authorizations and captures against orders.
package payments

import (
	"errors"
	"time"

	"shopcore/money"
)

// Status of a payment.
type Status string

const (
	StatusAuthorized Status = "authorized"
	StatusCaptured   Status = "captured"
	StatusVoided     Status = "voided"
	StatusRefunded   Status = "refunded"
)

var (
	ErrInvalidAmount        = errors.New("payments: amount must be positive")
	ErrInvalidState         = errors.New("payments: payment is not in a capturable state")
	ErrExceedsAuthorization = errors.New("payments: amount exceeds authorization")
)

// Payment is an authorization and, once captured, the captured amount.
type Payment struct {
	ID           string
	OrderID      string
	Processor    string
	Amount       money.Cents // authorized amount
	Captured     money.Cents
	Status       Status
	AuthorizedAt time.Time
	CapturedAt   time.Time
}

// Clone returns a copy.
func (p *Payment) Clone() *Payment {
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

// Outstanding is the authorized amount not yet captured.
func (p *Payment) Outstanding() money.Cents { return p.Amount - p.Captured }
