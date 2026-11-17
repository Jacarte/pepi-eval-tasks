package customers

import (
	"sort"
	"time"

	"shopcore/money"
)

// CustomerSummary aggregates a batch of customers for the reports package.
type CustomerSummary struct {
	Count              int
	ByStatus           map[CustomerStatus]int
	TotalLifetimeValue money.Cents
	Earliest           time.Time
	Latest             time.Time
}

// SummarizeCustomers folds items into a CustomerSummary. Nil entries are skipped.
func SummarizeCustomers(items []*Customer) CustomerSummary {
	sum := CustomerSummary{ByStatus: make(map[CustomerStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalLifetimeValue += v.LifetimeValue
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopCustomersByLifetimeValue returns the n items with the largest lifetime value, largest first.
// Ties keep their input order.
func TopCustomersByLifetimeValue(items []*Customer, n int) []*Customer {
	out := append([]*Customer(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].LifetimeValue > out[j].LifetimeValue })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupCustomersByEmail buckets items by email.
func GroupCustomersByEmail(items []*Customer) map[string][]*Customer {
	out := make(map[string][]*Customer)
	for _, v := range items {
		out[v.Email] = append(out[v.Email], v)
	}
	return out
}

// CustomersCreatedBetween keeps items created in [from, to).
func CustomersCreatedBetween(items []*Customer, from, to time.Time) []*Customer {
	var out []*Customer
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
