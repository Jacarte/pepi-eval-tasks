package tax

import (
	"encoding/csv"
	"io"
	"time"
)

// ExemptionSummary aggregates a batch of exemptions for the reports package.
type ExemptionSummary struct {
	Count    int
	ByStatus map[ExemptionStatus]int
	Earliest time.Time
	Latest   time.Time
}

// SummarizeExemptions folds items into a ExemptionSummary. Nil entries are skipped.
func SummarizeExemptions(items []*Exemption) ExemptionSummary {
	sum := ExemptionSummary{ByStatus: make(map[ExemptionStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// GroupExemptionsByCustomerID buckets items by customer id.
func GroupExemptionsByCustomerID(items []*Exemption) map[string][]*Exemption {
	out := make(map[string][]*Exemption)
	for _, v := range items {
		out[v.CustomerID] = append(out[v.CustomerID], v)
	}
	return out
}

// ExemptionsCreatedBetween keeps items created in [from, to).
func ExemptionsCreatedBetween(items []*Exemption, from, to time.Time) []*Exemption {
	var out []*Exemption
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteExemptionsCSV writes items as CSV with a header row.
// FIXME: this should probably live in a shared package.
func WriteExemptionsCSV(out io.Writer, items []*Exemption) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "customer_id", "certificate", "region", "expires_at", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.CustomerID,
			v.Certificate,
			v.Region,
			fmtTime(v.ExpiresAt),
			v.Status.String(),
			fmtTime(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
