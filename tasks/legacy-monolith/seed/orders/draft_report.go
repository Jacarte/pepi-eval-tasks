package orders

import (
	"fmt"
	"sort"
	"time"

	"shopcore/money"
)

// DraftSummary aggregates a batch of drafts for the reports package.
type DraftSummary struct {
	Count      int
	ByStatus   map[DraftStatus]int
	TotalItems int64
	TotalTotal money.Cents
	Earliest   time.Time
	Latest     time.Time
}

// SummarizeDrafts folds items into a DraftSummary. Nil entries are skipped.
func SummarizeDrafts(items []*Draft) DraftSummary {
	sum := DraftSummary{ByStatus: make(map[DraftStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalItems += int64(v.Items)
		sum.TotalTotal += v.Total
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopDraftsByItems returns the n items with the largest items, largest first.
// Ties keep their input order.
func TopDraftsByItems(items []*Draft, n int) []*Draft {
	out := append([]*Draft(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Items > out[j].Items })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupDraftsByCustomerID buckets items by customer id.
func GroupDraftsByCustomerID(items []*Draft) map[string][]*Draft {
	out := make(map[string][]*Draft)
	for _, v := range items {
		out[v.CustomerID] = append(out[v.CustomerID], v)
	}
	return out
}

// DraftsCreatedBetween keeps items created in [from, to).
func DraftsCreatedBetween(items []*Draft, from, to time.Time) []*Draft {
	var out []*Draft
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// DiffDraft lists the fields that differ between a and b.
func DiffDraft(a, b *Draft) []FieldChange {
	var out []FieldChange
	if a.CustomerID != b.CustomerID {
		out = append(out, FieldChange{"customer_id", fmt.Sprint(a.CustomerID), fmt.Sprint(b.CustomerID)})
	}
	if a.Items != b.Items {
		out = append(out, FieldChange{"items", fmt.Sprint(a.Items), fmt.Sprint(b.Items)})
	}
	if a.Note != b.Note {
		out = append(out, FieldChange{"note", fmt.Sprint(a.Note), fmt.Sprint(b.Note)})
	}
	if a.Total != b.Total {
		out = append(out, FieldChange{"total", fmt.Sprint(a.Total), fmt.Sprint(b.Total)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
