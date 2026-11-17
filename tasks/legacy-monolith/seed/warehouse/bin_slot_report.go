package warehouse

import (
	"fmt"
	"sort"
	"time"
)

// BinSlotSummary aggregates a batch of bin slots for the reports package.
type BinSlotSummary struct {
	Count         int
	ByStatus      map[BinSlotStatus]int
	TotalAisle    int64
	TotalLevel    int64
	TotalCapacity int64
	Earliest      time.Time
	Latest        time.Time
}

// SummarizeBinSlots folds items into a BinSlotSummary. Nil entries are skipped.
func SummarizeBinSlots(items []*BinSlot) BinSlotSummary {
	sum := BinSlotSummary{ByStatus: make(map[BinSlotStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalAisle += int64(v.Aisle)
		sum.TotalLevel += int64(v.Level)
		sum.TotalCapacity += int64(v.Capacity)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopBinSlotsByAisle returns the n items with the largest aisle, largest first.
// Ties keep their input order.
func TopBinSlotsByAisle(items []*BinSlot, n int) []*BinSlot {
	out := append([]*BinSlot(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Aisle > out[j].Aisle })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupBinSlotsByZone buckets items by zone.
func GroupBinSlotsByZone(items []*BinSlot) map[string][]*BinSlot {
	out := make(map[string][]*BinSlot)
	for _, v := range items {
		out[v.Zone] = append(out[v.Zone], v)
	}
	return out
}

// BinSlotsCreatedBetween keeps items created in [from, to).
func BinSlotsCreatedBetween(items []*BinSlot, from, to time.Time) []*BinSlot {
	var out []*BinSlot
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// DiffBinSlot lists the fields that differ between a and b.
func DiffBinSlot(a, b *BinSlot) []FieldChange {
	var out []FieldChange
	if a.Zone != b.Zone {
		out = append(out, FieldChange{"zone", fmt.Sprint(a.Zone), fmt.Sprint(b.Zone)})
	}
	if a.Aisle != b.Aisle {
		out = append(out, FieldChange{"aisle", fmt.Sprint(a.Aisle), fmt.Sprint(b.Aisle)})
	}
	if a.Level != b.Level {
		out = append(out, FieldChange{"level", fmt.Sprint(a.Level), fmt.Sprint(b.Level)})
	}
	if a.SKU != b.SKU {
		out = append(out, FieldChange{"sku", fmt.Sprint(a.SKU), fmt.Sprint(b.SKU)})
	}
	if a.Capacity != b.Capacity {
		out = append(out, FieldChange{"capacity", fmt.Sprint(a.Capacity), fmt.Sprint(b.Capacity)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
