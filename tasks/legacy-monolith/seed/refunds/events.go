package refunds

import (
	"context"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

// Refund event names.
const evRefundSettled = "refund.settled"

// record emits a refund event. Refund events always name the payment and the
// reason code: the chargeback desk searches the trail by payment.
func (s *Service) record(ctx context.Context, name string, r *Refund) error {
	return s.audit.Record(ctx, audit.Event{
		Name:      name,
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   r.ID,
		Attrs:     map[string]string{"payment_id": r.PaymentID, "reason": string(r.Reason)},
	})
}
