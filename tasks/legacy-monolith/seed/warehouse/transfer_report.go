package warehouse

import (
	"fmt"
	"sort"
	"time"
)

// TransferSummary aggregates a batch of transfers for the reports package.
type TransferSummary struct {
	Count    int
	ByStatus map[TransferStatus]int
	TotalQty int64
	Earliest time.Time
	Latest   time.Time
}

// SummarizeTransfers folds items into a TransferSummary. Nil entries are skipped.
func SummarizeTransfers(items []*Transfer) TransferSummary {
	sum := TransferSummary{ByStatus: make(map[TransferStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalQty += int64(v.Qty)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopTransfersByQty returns the n items with the largest qty, largest first.
// Ties keep their input order.
func TopTransfersByQty(items []*Transfer, n int) []*Transfer {
	out := append([]*Transfer(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Qty > out[j].Qty })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupTransfersByFromBin buckets items by from bin.
func GroupTransfersByFromBin(items []*Transfer) map[string][]*Transfer {
	out := make(map[string][]*Transfer)
	for _, v := range items {
		out[v.FromBin] = append(out[v.FromBin], v)
	}
	return out
}

// TransfersCreatedBetween keeps items created in [from, to).
func TransfersCreatedBetween(items []*Transfer, from, to time.Time) []*Transfer {
	var out []*Transfer
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// DiffTransfer lists the fields that differ between a and b.
func DiffTransfer(a, b *Transfer) []FieldChange {
	var out []FieldChange
	if a.FromBin != b.FromBin {
		out = append(out, FieldChange{"from_bin", fmt.Sprint(a.FromBin), fmt.Sprint(b.FromBin)})
	}
	if a.ToBin != b.ToBin {
		out = append(out, FieldChange{"to_bin", fmt.Sprint(a.ToBin), fmt.Sprint(b.ToBin)})
	}
	if a.SKU != b.SKU {
		out = append(out, FieldChange{"sku", fmt.Sprint(a.SKU), fmt.Sprint(b.SKU)})
	}
	if a.Qty != b.Qty {
		out = append(out, FieldChange{"qty", fmt.Sprint(a.Qty), fmt.Sprint(b.Qty)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
