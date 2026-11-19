// Package payments charges invoice totals through a payment gateway.
package payments

import (
	"errors"
	"fmt"

	"shopbill/internal/billing"
)

// Gateway accepts a charge in cents and returns a charge id.
type Gateway interface {
	Charge(cents int64) (string, error)
}

// ChargeInvoice charges the invoice total. Refund-only invoices are not charged.
func ChargeInvoice(g Gateway, inv billing.Invoice) (string, error) {
	total := inv.Totals().TotalCents
	if total <= 0 {
		return "", errors.New("payments: nothing to charge")
	}
	id, err := g.Charge(total)
	if err != nil {
		return "", fmt.Errorf("payments: invoice %s: %w", inv.ID, err)
	}
	return id, nil
}
