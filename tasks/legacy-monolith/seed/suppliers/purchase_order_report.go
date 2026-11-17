package suppliers

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"time"

	"shopcore/money"
)

// PurchaseOrderSummary aggregates a batch of purchase orders for the reports package.
type PurchaseOrderSummary struct {
	Count      int
	ByStatus   map[PurchaseOrderStatus]int
	TotalTotal money.Cents
	TotalLines int64
	Earliest   time.Time
	Latest     time.Time
}

// SummarizePurchaseOrders folds items into a PurchaseOrderSummary. Nil entries are skipped.
func SummarizePurchaseOrders(items []*PurchaseOrder) PurchaseOrderSummary {
	sum := PurchaseOrderSummary{ByStatus: make(map[PurchaseOrderStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalTotal += v.Total
		sum.TotalLines += int64(v.Lines)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopPurchaseOrdersByTotal returns the n items with the largest total, largest first.
// Ties keep their input order.
func TopPurchaseOrdersByTotal(items []*PurchaseOrder, n int) []*PurchaseOrder {
	out := append([]*PurchaseOrder(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupPurchaseOrdersBySupplierID buckets items by supplier id.
func GroupPurchaseOrdersBySupplierID(items []*PurchaseOrder) map[string][]*PurchaseOrder {
	out := make(map[string][]*PurchaseOrder)
	for _, v := range items {
		out[v.SupplierID] = append(out[v.SupplierID], v)
	}
	return out
}

// PurchaseOrdersCreatedBetween keeps items created in [from, to).
func PurchaseOrdersCreatedBetween(items []*PurchaseOrder, from, to time.Time) []*PurchaseOrder {
	var out []*PurchaseOrder
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WritePurchaseOrdersCSV writes items as CSV with a header row.
func WritePurchaseOrdersCSV(out io.Writer, items []*PurchaseOrder) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "supplier_id", "total", "lines", "expected_at", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.SupplierID,
			money.Cents(v.Total).String(),
			strconv.Itoa(v.Lines),
			formatTime(v.ExpectedAt),
			v.Status.String(),
			formatTime(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
