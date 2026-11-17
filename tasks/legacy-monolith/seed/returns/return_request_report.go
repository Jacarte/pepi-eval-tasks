package returns

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

// ReturnRequestSummary aggregates a batch of return requests for the reports package.
type ReturnRequestSummary struct {
	Count    int
	ByStatus map[ReturnRequestStatus]int
	TotalQty int64
	Earliest time.Time
	Latest   time.Time
}

// SummarizeReturnRequests folds items into a ReturnRequestSummary. Nil entries are skipped.
func SummarizeReturnRequests(items []*ReturnRequest) ReturnRequestSummary {
	sum := ReturnRequestSummary{ByStatus: make(map[ReturnRequestStatus]int)}
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

// TopReturnRequestsByQty returns the n items with the largest qty, largest first.
// Ties keep their input order.
func TopReturnRequestsByQty(items []*ReturnRequest, n int) []*ReturnRequest {
	out := append([]*ReturnRequest(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Qty > out[j].Qty })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupReturnRequestsByOrderID buckets items by order id.
func GroupReturnRequestsByOrderID(items []*ReturnRequest) map[string][]*ReturnRequest {
	out := make(map[string][]*ReturnRequest)
	for _, v := range items {
		out[v.OrderID] = append(out[v.OrderID], v)
	}
	return out
}

// ReturnRequestsCreatedBetween keeps items created in [from, to).
func ReturnRequestsCreatedBetween(items []*ReturnRequest, from, to time.Time) []*ReturnRequest {
	var out []*ReturnRequest
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteReturnRequestsCSV writes items as CSV with a header row.
func WriteReturnRequestsCSV(out io.Writer, items []*ReturnRequest) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "order_id", "sku", "qty", "reason", "inspector", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.OrderID,
			v.SKU,
			strconv.Itoa(v.Qty),
			v.Reason,
			v.Inspector,
			v.Status.String(),
			timeString(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DiffReturnRequest lists the fields that differ between a and b.
func DiffReturnRequest(a, b *ReturnRequest) []FieldChange {
	var out []FieldChange
	if a.OrderID != b.OrderID {
		out = append(out, FieldChange{"order_id", fmt.Sprint(a.OrderID), fmt.Sprint(b.OrderID)})
	}
	if a.SKU != b.SKU {
		out = append(out, FieldChange{"sku", fmt.Sprint(a.SKU), fmt.Sprint(b.SKU)})
	}
	if a.Qty != b.Qty {
		out = append(out, FieldChange{"qty", fmt.Sprint(a.Qty), fmt.Sprint(b.Qty)})
	}
	if a.Reason != b.Reason {
		out = append(out, FieldChange{"reason", fmt.Sprint(a.Reason), fmt.Sprint(b.Reason)})
	}
	if a.Inspector != b.Inspector {
		out = append(out, FieldChange{"inspector", fmt.Sprint(a.Inspector), fmt.Sprint(b.Inspector)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
