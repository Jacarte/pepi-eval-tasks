package fraud

import (
	"sort"
	"time"
)

// RuleSummary aggregates a batch of rules for the reports package.
type RuleSummary struct {
	Count       int
	ByStatus    map[RuleStatus]int
	TotalWeight int64
	Earliest    time.Time
	Latest      time.Time
}

// SummarizeRules folds items into a RuleSummary. Nil entries are skipped.
func SummarizeRules(items []*Rule) RuleSummary {
	sum := RuleSummary{ByStatus: make(map[RuleStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalWeight += int64(v.Weight)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopRulesByWeight returns the n items with the largest weight, largest first.
// Ties keep their input order.
func TopRulesByWeight(items []*Rule, n int) []*Rule {
	out := append([]*Rule(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Weight > out[j].Weight })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupRulesByName buckets items by name.
func GroupRulesByName(items []*Rule) map[string][]*Rule {
	out := make(map[string][]*Rule)
	for _, v := range items {
		out[v.Name] = append(out[v.Name], v)
	}
	return out
}

// RulesCreatedBetween keeps items created in [from, to).
func RulesCreatedBetween(items []*Rule, from, to time.Time) []*Rule {
	var out []*Rule
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
