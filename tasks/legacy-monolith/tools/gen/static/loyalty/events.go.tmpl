package loyalty

import (
	"context"
	"strconv"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

const evAccountAccrued = "account.accrued"

// event builds a loyalty audit event. Loyalty events are filed under the
// customer, not the account id: the support tool looks them up from the
// customer page and accounts can be re-opened under a new id after a merge.
func (s *Service) event(ctx context.Context, name string, a *Account, points int64) audit.Event {
	return audit.Event{
		Name:      name,
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   a.CustomerID,
		Attrs:     map[string]string{"account_id": a.ID, "points": strconv.FormatInt(points, 10)},
	}
}
