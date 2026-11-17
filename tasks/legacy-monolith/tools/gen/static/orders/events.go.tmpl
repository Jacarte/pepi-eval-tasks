package orders

import (
	"context"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

// Order event names.
const (
	evOrderCancelled = "order.cancelled"
	evOrderFulfilled = "order.fulfilled"
)

// emit records an order event. Support filters the order timeline by
// customer, so every order event carries customer_id; extra attributes are
// merged on top.
func (s *Service) emit(ctx context.Context, name string, o *Order, extra map[string]string) error {
	attrs := map[string]string{"customer_id": o.CustomerID}
	for k, v := range extra {
		attrs[k] = v
	}
	return s.audit.Record(ctx, audit.Event{
		Name:      name,
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   o.ID,
		Attrs:     attrs,
	})
}
