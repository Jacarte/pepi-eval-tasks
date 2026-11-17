package payments

import (
	"context"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

const (
	evPaymentAuthorized = "payment.authorized"
	evPaymentVoided     = "payment.voided"
)

// paymentEvent builds the audit event for a payment state change. Finance
// reconciles payment events against the processor report, which is keyed by
// order and processor, so both are always attached.
func paymentEvent(ctx context.Context, name string, p *Payment) audit.Event {
	return audit.Event{
		Name:      name,
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   p.ID,
		Attrs:     map[string]string{"order_id": p.OrderID, "processor": p.Processor},
	}
}
