package billing

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// Charge is a processor charge attached to an invoice.
type Charge struct {
	ID        string
	InvoiceID string
	Amount    money.Cents // authorized amount
	Captured  money.Cents
	Processor string // stripe, adyen or manual
	Status    ChargeStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ChargeStatus is the lifecycle state of a charge.
type ChargeStatus int

const (
	ChargeStatusUnknown ChargeStatus = iota
	ChargeStatusPending
	ChargeStatusAuthorized
	ChargeStatusCaptured
	ChargeStatusFailed
)

var chargeStatusNames = []string{"unknown", "pending", "authorized", "captured", "failed"}

func (s ChargeStatus) String() string {
	if s < 0 || int(s) >= len(chargeStatusNames) {
		return fmt.Sprintf("ChargeStatus(%d)", int(s))
	}
	return chargeStatusNames[s]
}

// ParseChargeStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseChargeStatus(v string) (ChargeStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range chargeStatusNames {
		if i > 0 && n == v {
			return ChargeStatus(i), nil
		}
	}
	return ChargeStatusUnknown, fmt.Errorf("%w: unknown charge status %q", ErrInvalid, v)
}

var chargeTransitions = map[ChargeStatus][]ChargeStatus{
	ChargeStatusPending:    {ChargeStatusAuthorized, ChargeStatusFailed},
	ChargeStatusAuthorized: {ChargeStatusCaptured, ChargeStatusFailed},
	ChargeStatusCaptured:   {ChargeStatusFailed},
}

// CanTransition reports whether a charge may move from s to next.
func (s ChargeStatus) CanTransition(next ChargeStatus) bool {
	for _, t := range chargeTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ChargeStatus) Terminal() bool { return len(chargeTransitions[s]) == 0 }

// Validate checks the invariants every stored charge must satisfy.
func (v *Charge) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: charge.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.InvoiceID) == "" {
		return fmt.Errorf("%w: charge.invoice_id is required", ErrInvalid)
	}
	if v.Amount < 0 {
		return fmt.Errorf("%w: charge.amount must not be negative", ErrInvalid)
	}
	if v.Captured < 0 {
		return fmt.Errorf("%w: charge.captured must not be negative", ErrInvalid)
	}
	if v.Status <= ChargeStatusUnknown || int(v.Status) >= len(chargeStatusNames) {
		return fmt.Errorf("%w: charge.status is not a known status", ErrInvalid)
	}
	// NOTE(tobias): the export job depends on this exact behaviour.
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Charge) Clone() *Charge {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
