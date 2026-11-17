// Package loyalty runs the points programme: accrual on orders, redemption at
// checkout, and tier calculation.
package loyalty

import (
	"errors"
	"time"
)

var (
	ErrInvalidPoints      = errors.New("loyalty: points must be positive")
	ErrInsufficientPoints = errors.New("loyalty: insufficient points")
)

// Account is a customer's points balance.
type Account struct {
	ID         string
	CustomerID string
	Balance    int64
	Lifetime   int64 // total ever accrued; drives the tier
	TierName   string
	UpdatedAt  time.Time
}

func (a *Account) clone() *Account {
	cp := *a
	return &cp
}

// tierFor picks the highest tier whose threshold lifetime reaches.
func tierFor(lifetime int64) string {
	switch {
	case lifetime >= 50000:
		return "gold"
	case lifetime >= 10000:
		return "silver"
	default:
		return "bronze"
	}
}
