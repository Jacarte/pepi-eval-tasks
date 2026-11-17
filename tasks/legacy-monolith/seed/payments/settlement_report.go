package payments

import (
	"sort"
	"time"

	"shopcore/money"
)

// SettlementSummary aggregates a batch of settlements for the reports package.
type SettlementSummary struct {
	Count       int
	ByStatus    map[SettlementStatus]int
	TotalAmount money.Cents
	TotalItems  int64
	Earliest    time.Time
	Latest      time.Time
}

// SummarizeSettlements folds items into a SettlementSummary. Nil entries are skipped.
func SummarizeSettlements(items []*Settlement) SettlementSummary {
	sum := SettlementSummary{ByStatus: make(map[SettlementStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalAmount += v.Amount
		sum.TotalItems += int64(v.Items)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopSettlementsByAmount returns the n items with the largest amount, largest first.
// Ties keep their input order.
func TopSettlementsByAmount(items []*Settlement, n int) []*Settlement {
	out := append([]*Settlement(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Amount > out[j].Amount })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupSettlementsByBatch buckets items by batch.
func GroupSettlementsByBatch(items []*Settlement) map[string][]*Settlement {
	out := make(map[string][]*Settlement)
	for _, v := range items {
		out[v.Batch] = append(out[v.Batch], v)
	}
	return out
}

// SettlementsCreatedBetween keeps items created in [from, to).
func SettlementsCreatedBetween(items []*Settlement, from, to time.Time) []*Settlement {
	var out []*Settlement
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
