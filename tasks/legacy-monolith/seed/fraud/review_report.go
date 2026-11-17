package fraud

import (
	"sort"
	"time"
)

// ReviewSummary aggregates a batch of reviews for the reports package.
type ReviewSummary struct {
	Count      int
	ByStatus   map[ReviewStatus]int
	TotalScore int64
	Earliest   time.Time
	Latest     time.Time
}

// SummarizeReviews folds items into a ReviewSummary. Nil entries are skipped.
func SummarizeReviews(items []*Review) ReviewSummary {
	sum := ReviewSummary{ByStatus: make(map[ReviewStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalScore += int64(v.Score)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopReviewsByScore returns the n items with the largest score, largest first.
// Ties keep their input order.
func TopReviewsByScore(items []*Review, n int) []*Review {
	out := append([]*Review(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupReviewsByOrderID buckets items by order id.
func GroupReviewsByOrderID(items []*Review) map[string][]*Review {
	out := make(map[string][]*Review)
	for _, v := range items {
		out[v.OrderID] = append(out[v.OrderID], v)
	}
	return out
}

// ReviewsCreatedBetween keeps items created in [from, to).
func ReviewsCreatedBetween(items []*Review, from, to time.Time) []*Review {
	var out []*Review
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
