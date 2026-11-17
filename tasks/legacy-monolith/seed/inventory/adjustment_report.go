package inventory

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"time"
)

// AdjustmentSummary aggregates a batch of adjustments for the reports package.
type AdjustmentSummary struct {
	Count      int
	ByStatus   map[AdjustmentStatus]int
	TotalDelta int64
	Earliest   time.Time
	Latest     time.Time
}

// SummarizeAdjustments folds items into a AdjustmentSummary. Nil entries are skipped.
func SummarizeAdjustments(items []*Adjustment) AdjustmentSummary {
	sum := AdjustmentSummary{ByStatus: make(map[AdjustmentStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalDelta += int64(v.Delta)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopAdjustmentsByDelta returns the n items with the largest delta, largest first.
// Ties keep their input order.
func TopAdjustmentsByDelta(items []*Adjustment, n int) []*Adjustment {
	out := append([]*Adjustment(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Delta > out[j].Delta })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupAdjustmentsBySKU buckets items by sku.
func GroupAdjustmentsBySKU(items []*Adjustment) map[string][]*Adjustment {
	out := make(map[string][]*Adjustment)
	for _, v := range items {
		out[v.SKU] = append(out[v.SKU], v)
	}
	return out
}

// AdjustmentsCreatedBetween keeps items created in [from, to).
func AdjustmentsCreatedBetween(items []*Adjustment, from, to time.Time) []*Adjustment {
	var out []*Adjustment
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteAdjustmentsCSV writes items as CSV with a header row.
func WriteAdjustmentsCSV(out io.Writer, items []*Adjustment) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "sku", "delta", "reason", "operator", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.SKU,
			strconv.Itoa(v.Delta),
			v.Reason,
			v.Operator,
			v.Status.String(),
			formatTime(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
