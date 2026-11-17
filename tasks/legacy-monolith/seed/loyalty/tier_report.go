package loyalty

import (
	"sort"
	"time"
)

// TierSummary aggregates a batch of tiers for the reports package.
type TierSummary struct {
	Count           int
	ByStatus        map[TierStatus]int
	TotalThreshold  int64
	TotalMultiplier int64
	Earliest        time.Time
	Latest          time.Time
}

// SummarizeTiers folds items into a TierSummary. Nil entries are skipped.
func SummarizeTiers(items []*Tier) TierSummary {
	sum := TierSummary{ByStatus: make(map[TierStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalThreshold += v.Threshold
		sum.TotalMultiplier += int64(v.Multiplier)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopTiersByThreshold returns the n items with the largest threshold, largest first.
// Ties keep their input order.
func TopTiersByThreshold(items []*Tier, n int) []*Tier {
	out := append([]*Tier(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Threshold > out[j].Threshold })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupTiersByName buckets items by name.
func GroupTiersByName(items []*Tier) map[string][]*Tier {
	out := make(map[string][]*Tier)
	for _, v := range items {
		out[v.Name] = append(out[v.Name], v)
	}
	return out
}

// TiersCreatedBetween keeps items created in [from, to).
func TiersCreatedBetween(items []*Tier, from, to time.Time) []*Tier {
	var out []*Tier
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
