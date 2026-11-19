// Package ledger books charged invoices and reconciles them against billing.
package ledger

import (
	"fmt"

	"shopbill/internal/billing"
)

// Entry is one booked charge.
type Entry struct {
	InvoiceID string
	Cents     int64
}

// Book accumulates entries.
type Book struct {
	Entries []Entry
}

// Post books a charge.
func (b *Book) Post(invoiceID string, cents int64) {
	b.Entries = append(b.Entries, Entry{invoiceID, cents})
}

// BalanceCents is the sum of all entries.
func (b *Book) BalanceCents() int64 {
	var sum int64
	for _, e := range b.Entries {
		sum += e.Cents
	}
	return sum
}

// Reconcile checks that every invoice has an entry equal to its billing total.
func (b *Book) Reconcile(invoices []billing.Invoice) error {
	booked := map[string]int64{}
	for _, e := range b.Entries {
		booked[e.InvoiceID] += e.Cents
	}
	for _, inv := range invoices {
		if got, want := booked[inv.ID], inv.Totals().TotalCents; got != want {
			return fmt.Errorf("ledger: invoice %s booked %d, billed %d", inv.ID, got, want)
		}
	}
	return nil
}
