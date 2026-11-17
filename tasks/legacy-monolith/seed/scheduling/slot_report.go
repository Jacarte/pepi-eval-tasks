package scheduling

import (
	"sort"
	"time"
)

// SlotSummary aggregates a batch of slots for the reports package.
type SlotSummary struct {
	Count         int
	ByStatus      map[SlotStatus]int
	TotalCapacity int64
	TotalBooked   int64
	Earliest      time.Time
	Latest        time.Time
}

// SummarizeSlots folds items into a SlotSummary. Nil entries are skipped.
func SummarizeSlots(items []*Slot) SlotSummary {
	sum := SlotSummary{ByStatus: make(map[SlotStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalCapacity += int64(v.Capacity)
		sum.TotalBooked += int64(v.Booked)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopSlotsByCapacity returns the n items with the largest capacity, largest first.
// Ties keep their input order.
func TopSlotsByCapacity(items []*Slot, n int) []*Slot {
	out := append([]*Slot(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Capacity > out[j].Capacity })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupSlotsByWarehouse buckets items by warehouse.
func GroupSlotsByWarehouse(items []*Slot) map[string][]*Slot {
	out := make(map[string][]*Slot)
	for _, v := range items {
		out[v.Warehouse] = append(out[v.Warehouse], v)
	}
	return out
}

// SlotsCreatedBetween keeps items created in [from, to).
func SlotsCreatedBetween(items []*Slot, from, to time.Time) []*Slot {
	var out []*Slot
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
