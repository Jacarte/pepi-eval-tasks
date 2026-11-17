// Package fraud scores orders and runs the manual review queue.
package fraud

import (
	"context"
	"errors"
	"strings"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

var (
	ErrNotOpen   = errors.New("fraud: review is not open")
	ErrNoAnalyst = errors.New("fraud: analyst is required")
)

const defaultCountryRisk = 60

// Signals are the order facts the scorer looks at.
type Signals struct {
	Country        string
	AmountCents    int64
	AccountAgeDays int
	FailedPayments int
	AddressMatch   bool
}

// Score returns a 0-1000 risk score. Rules come from the store; the country
// table is the baseline.
func Score(sig Signals, rules []*Rule) int {
	score, ok := countryRisk[strings.ToUpper(sig.Country)]
	if !ok {
		score = defaultCountryRisk
	}
	for _, r := range rules {
		if !r.Enabled || r.Status != RuleStatusLive {
			continue
		}
		if matches(r.Expression, sig) {
			score += r.Weight
		}
	}
	if score > 1000 {
		score = 1000
	}
	return score
}

// matches evaluates the tiny legacy rule DSL: "amount>N", "age<N",
// "failed>N" and "nomatch". Anything else never matches.
func matches(expr string, sig Signals) bool {
	expr = strings.ReplaceAll(strings.ToLower(expr), " ", "")
	switch {
	case expr == "nomatch":
		return !sig.AddressMatch
	case strings.HasPrefix(expr, "amount>"):
		return sig.AmountCents > atoi(expr[len("amount>"):])
	case strings.HasPrefix(expr, "age<"):
		return int64(sig.AccountAgeDays) < atoi(expr[len("age<"):])
	case strings.HasPrefix(expr, "failed>"):
		return int64(sig.FailedPayments) > atoi(expr[len("failed>"):])
	}
	return false
}

func atoi(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 1 << 62
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

// Desk is the manual review queue.
type Desk struct {
	reviews ReviewStore
	audit   audit.Recorder
	clock   clock.Clock
}

func NewDesk(reviews ReviewStore, rec audit.Recorder, clk clock.Clock) *Desk {
	return &Desk{reviews: reviews, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk)}
}

// Escalate hands an open review to a senior analyst.
func (d *Desk) Escalate(ctx context.Context, reviewID, analyst string) (*Review, error) {
	const op = "fraud.escalate"
	if analyst == "" {
		return nil, errs.Wrap(op, ErrNoAnalyst)
	}
	rv, err := d.reviews.Find(ctx, reviewID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if rv.Status != ReviewStatusOpen {
		return nil, errs.Wrap(op, ErrNotOpen)
	}
	rv.Status = ReviewStatusEscalated
	rv.Analyst = analyst
	rv.UpdatedAt = d.clock.Now()
	if err := d.reviews.Save(ctx, rv); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := d.audit.Record(ctx, audit.Event{
		Name:      "review.escalated",
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   rv.ID,
		Attrs:     map[string]string{"analyst": analyst, "order_id": rv.OrderID},
	}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return rv, nil
}

// Clear closes a review as legitimate so the order can ship.
func (d *Desk) Clear(ctx context.Context, reviewID string) (*Review, error) {
	const op = "fraud.clear"
	rv, err := d.reviews.Find(ctx, reviewID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if rv.Status != ReviewStatusOpen && rv.Status != ReviewStatusEscalated {
		return nil, errs.Wrap(op, ErrNotOpen)
	}
	rv.Status = ReviewStatusCleared
	rv.UpdatedAt = d.clock.Now()
	if err := d.reviews.Save(ctx, rv); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := d.audit.Record(ctx, audit.Event{
		Name:      "review.cleared",
		RequestID: ctxkeys.RequestIDFrom(ctx),
		Subject:   rv.ID,
	}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return rv, nil
}
