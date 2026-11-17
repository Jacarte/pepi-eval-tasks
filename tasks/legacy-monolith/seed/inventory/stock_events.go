package inventory

import (
	"context"
	"strconv"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

const (
	evStockReceived = "stock_item.received"
	evStockReleased = "stock_item.released"
)

// stockEvent describes a stock movement. qty is the number of units the
// operation moved, not the resulting position; warehouse lets the WMS match
// the event to a site.
func stockEvent(ctx context.Context, name string, it *StockItem, qty int) audit.Event {
	return audit.Event{
		Name:      name,
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   it.ID,
		Attrs:     map[string]string{"warehouse": it.Warehouse, "qty": strconv.Itoa(qty)},
	}
}
