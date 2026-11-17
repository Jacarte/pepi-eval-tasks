package catalog

import (
	"sort"
	"time"

	"shopcore/money"
)

// ProductListingSummary aggregates a batch of product listings for the reports package.
type ProductListingSummary struct {
	Count      int
	ByStatus   map[ProductListingStatus]int
	TotalPrice money.Cents
	Earliest   time.Time
	Latest     time.Time
}

// SummarizeProductListings folds items into a ProductListingSummary. Nil entries are skipped.
func SummarizeProductListings(items []*ProductListing) ProductListingSummary {
	sum := ProductListingSummary{ByStatus: make(map[ProductListingStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalPrice += v.Price
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopProductListingsByPrice returns the n items with the largest price, largest first.
// Ties keep their input order.
func TopProductListingsByPrice(items []*ProductListing, n int) []*ProductListing {
	out := append([]*ProductListing(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Price > out[j].Price })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupProductListingsBySKU buckets items by sku.
func GroupProductListingsBySKU(items []*ProductListing) map[string][]*ProductListing {
	out := make(map[string][]*ProductListing)
	for _, v := range items {
		out[v.SKU] = append(out[v.SKU], v)
	}
	return out
}

// ProductListingsCreatedBetween keeps items created in [from, to).
func ProductListingsCreatedBetween(items []*ProductListing, from, to time.Time) []*ProductListing {
	var out []*ProductListing
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
