// Package notify formats customer messages.
package notify

import (
	"fmt"

	"shopbill/internal/billing"
)

// Cents formats an amount in cents as dollars, e.g. -1103 becomes "-$11.03".
func Cents(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s$%d.%02d", sign, c/100, c%100)
}

// Format is the one-line summary of an invoice.
func Format(inv billing.Invoice) string {
	t := inv.Totals()
	return fmt.Sprintf("Invoice %s: %s (tax %s)", inv.ID, Cents(t.TotalCents), Cents(t.TaxCents))
}
