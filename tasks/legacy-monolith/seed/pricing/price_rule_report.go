package pricing

import (
	"sort"
	"time"
)

// PriceRuleSummary aggregates a batch of price rules for the reports package.
type PriceRuleSummary struct {
	Count         int
	ByStatus      map[PriceRuleStatus]int
	TotalValue    int64
	TotalPriority int64
	Earliest      time.Time
	Latest        time.Time
}

// SummarizePriceRules folds items into a PriceRuleSummary. Nil entries are skipped.
func SummarizePriceRules(items []*PriceRule) PriceRuleSummary {
	sum := PriceRuleSummary{ByStatus: make(map[PriceRuleStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalValue += v.Value
		sum.TotalPriority += int64(v.Priority)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopPriceRulesByValue returns the n items with the largest value, largest first.
// Ties keep their input order.
func TopPriceRulesByValue(items []*PriceRule, n int) []*PriceRule {
	out := append([]*PriceRule(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Value > out[j].Value })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupPriceRulesBySKU buckets items by sku.
func GroupPriceRulesBySKU(items []*PriceRule) map[string][]*PriceRule {
	out := make(map[string][]*PriceRule)
	for _, v := range items {
		out[v.SKU] = append(out[v.SKU], v)
	}
	return out
}

// PriceRulesCreatedBetween keeps items created in [from, to).
func PriceRulesCreatedBetween(items []*PriceRule, from, to time.Time) []*PriceRule {
	var out []*PriceRule
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
