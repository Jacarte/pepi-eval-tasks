package shipping

import (
	"encoding/csv"
	"io"
	"sort"
	"strconv"
	"time"
)

// ShipmentSummary aggregates a batch of shipments for the reports package.
type ShipmentSummary struct {
	Count            int
	ByStatus         map[ShipmentStatus]int
	TotalWeightGrams int64
	Earliest         time.Time
	Latest           time.Time
}

// SummarizeShipments folds items into a ShipmentSummary. Nil entries are skipped.
func SummarizeShipments(items []*Shipment) ShipmentSummary {
	sum := ShipmentSummary{ByStatus: make(map[ShipmentStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalWeightGrams += int64(v.WeightGrams)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopShipmentsByWeightGrams returns the n items with the largest weight grams, largest first.
// Ties keep their input order.
func TopShipmentsByWeightGrams(items []*Shipment, n int) []*Shipment {
	out := append([]*Shipment(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].WeightGrams > out[j].WeightGrams })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupShipmentsByOrderID buckets items by order id.
func GroupShipmentsByOrderID(items []*Shipment) map[string][]*Shipment {
	out := make(map[string][]*Shipment)
	for _, v := range items {
		out[v.OrderID] = append(out[v.OrderID], v)
	}
	return out
}

// ShipmentsCreatedBetween keeps items created in [from, to).
func ShipmentsCreatedBetween(items []*Shipment, from, to time.Time) []*Shipment {
	var out []*Shipment
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteShipmentsCSV writes items as CSV with a header row.
func WriteShipmentsCSV(out io.Writer, items []*Shipment) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "order_id", "carrier", "tracking_no", "weight_grams", "service", "hold_reason", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.OrderID,
			v.Carrier,
			v.TrackingNo,
			strconv.Itoa(v.WeightGrams),
			v.Service,
			v.HoldReason,
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
