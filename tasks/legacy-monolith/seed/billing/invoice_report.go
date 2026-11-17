package billing

import (
	"fmt"
	"sort"
	"time"

	"shopcore/money"
)

// InvoiceSummary aggregates a batch of invoices for the reports package.
type InvoiceSummary struct {
	Count          int
	ByStatus       map[InvoiceStatus]int
	TotalAmountDue money.Cents
	TotalLines     int64
	Earliest       time.Time
	Latest         time.Time
}

// SummarizeInvoices folds items into a InvoiceSummary. Nil entries are skipped.
func SummarizeInvoices(items []*Invoice) InvoiceSummary {
	sum := InvoiceSummary{ByStatus: make(map[InvoiceStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalAmountDue += v.AmountDue
		sum.TotalLines += int64(v.Lines)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopInvoicesByAmountDue returns the n items with the largest amount due, largest first.
// Ties keep their input order.
func TopInvoicesByAmountDue(items []*Invoice, n int) []*Invoice {
	out := append([]*Invoice(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].AmountDue > out[j].AmountDue })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupInvoicesByCustomerID buckets items by customer id.
func GroupInvoicesByCustomerID(items []*Invoice) map[string][]*Invoice {
	out := make(map[string][]*Invoice)
	for _, v := range items {
		out[v.CustomerID] = append(out[v.CustomerID], v)
	}
	return out
}

// InvoicesCreatedBetween keeps items created in [from, to).
func InvoicesCreatedBetween(items []*Invoice, from, to time.Time) []*Invoice {
	var out []*Invoice
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// DiffInvoice lists the fields that differ between a and b.
func DiffInvoice(a, b *Invoice) []FieldChange {
	var out []FieldChange
	if a.CustomerID != b.CustomerID {
		out = append(out, FieldChange{"customer_id", fmt.Sprint(a.CustomerID), fmt.Sprint(b.CustomerID)})
	}
	if a.Number != b.Number {
		out = append(out, FieldChange{"number", fmt.Sprint(a.Number), fmt.Sprint(b.Number)})
	}
	if a.AmountDue != b.AmountDue {
		out = append(out, FieldChange{"amount_due", fmt.Sprint(a.AmountDue), fmt.Sprint(b.AmountDue)})
	}
	if a.Currency != b.Currency {
		out = append(out, FieldChange{"currency", fmt.Sprint(a.Currency), fmt.Sprint(b.Currency)})
	}
	if !a.DueAt.Equal(b.DueAt) {
		out = append(out, FieldChange{"due_at", timeString(a.DueAt), timeString(b.DueAt)})
	}
	if a.Lines != b.Lines {
		out = append(out, FieldChange{"lines", fmt.Sprint(a.Lines), fmt.Sprint(b.Lines)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
