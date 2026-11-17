package billing

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"time"

	"shopcore/money"
)

// ChargeSummary aggregates a batch of charges for the reports package.
type ChargeSummary struct {
	Count         int
	ByStatus      map[ChargeStatus]int
	TotalAmount   money.Cents
	TotalCaptured money.Cents
	Earliest      time.Time
	Latest        time.Time
}

// SummarizeCharges folds items into a ChargeSummary. Nil entries are skipped.
func SummarizeCharges(items []*Charge) ChargeSummary {
	sum := ChargeSummary{ByStatus: make(map[ChargeStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalAmount += v.Amount
		sum.TotalCaptured += v.Captured
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopChargesByAmount returns the n items with the largest amount, largest first.
// Ties keep their input order.
func TopChargesByAmount(items []*Charge, n int) []*Charge {
	out := append([]*Charge(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Amount > out[j].Amount })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupChargesByInvoiceID buckets items by invoice id.
func GroupChargesByInvoiceID(items []*Charge) map[string][]*Charge {
	out := make(map[string][]*Charge)
	for _, v := range items {
		out[v.InvoiceID] = append(out[v.InvoiceID], v)
	}
	return out
}

// ChargesCreatedBetween keeps items created in [from, to).
func ChargesCreatedBetween(items []*Charge, from, to time.Time) []*Charge {
	var out []*Charge
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteChargesCSV writes items as CSV with a header row.
func WriteChargesCSV(out io.Writer, items []*Charge) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "invoice_id", "amount", "captured", "processor", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.InvoiceID,
			money.Cents(v.Amount).String(),
			money.Cents(v.Captured).String(),
			v.Processor,
			v.Status.String(),
			timeString(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DiffCharge lists the fields that differ between a and b.
func DiffCharge(a, b *Charge) []FieldChange {
	var out []FieldChange
	if a.InvoiceID != b.InvoiceID {
		out = append(out, FieldChange{"invoice_id", fmt.Sprint(a.InvoiceID), fmt.Sprint(b.InvoiceID)})
	}
	if a.Amount != b.Amount {
		out = append(out, FieldChange{"amount", fmt.Sprint(a.Amount), fmt.Sprint(b.Amount)})
	}
	if a.Captured != b.Captured {
		out = append(out, FieldChange{"captured", fmt.Sprint(a.Captured), fmt.Sprint(b.Captured)})
	}
	if a.Processor != b.Processor {
		out = append(out, FieldChange{"processor", fmt.Sprint(a.Processor), fmt.Sprint(b.Processor)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
