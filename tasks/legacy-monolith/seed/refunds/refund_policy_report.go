package refunds

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"time"
)

// RefundPolicySummary aggregates a batch of refund policies for the reports package.
type RefundPolicySummary struct {
	Count              int
	ByStatus           map[RefundPolicyStatus]int
	TotalWindowDays    int64
	TotalRestockFeePct int64
	Earliest           time.Time
	Latest             time.Time
}

// SummarizeRefundPolicies folds items into a RefundPolicySummary. Nil entries are skipped.
func SummarizeRefundPolicies(items []*RefundPolicy) RefundPolicySummary {
	sum := RefundPolicySummary{ByStatus: make(map[RefundPolicyStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalWindowDays += int64(v.WindowDays)
		sum.TotalRestockFeePct += int64(v.RestockFeePct)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopRefundPoliciesByWindowDays returns the n items with the largest window days, largest first.
// Ties keep their input order.
func TopRefundPoliciesByWindowDays(items []*RefundPolicy, n int) []*RefundPolicy {
	out := append([]*RefundPolicy(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].WindowDays > out[j].WindowDays })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupRefundPoliciesByName buckets items by name.
func GroupRefundPoliciesByName(items []*RefundPolicy) map[string][]*RefundPolicy {
	out := make(map[string][]*RefundPolicy)
	for _, v := range items {
		out[v.Name] = append(out[v.Name], v)
	}
	return out
}

// RefundPoliciesCreatedBetween keeps items created in [from, to).
func RefundPoliciesCreatedBetween(items []*RefundPolicy, from, to time.Time) []*RefundPolicy {
	var out []*RefundPolicy
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteRefundPoliciesCSV writes items as CSV with a header row.
// TODO(lw): revisit after the 2021 data migration (OPS-1423).
func WriteRefundPoliciesCSV(out io.Writer, items []*RefundPolicy) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "name", "window_days", "restock_fee_pct", "requires_receipt", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Name,
			strconv.Itoa(v.WindowDays),
			strconv.Itoa(v.RestockFeePct),
			strconv.FormatBool(v.RequiresReceipt),
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
