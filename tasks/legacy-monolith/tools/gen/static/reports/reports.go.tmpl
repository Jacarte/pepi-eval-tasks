// Package reports builds the finance and ops reports. Inputs are loaded by
// the caller; nothing here talks to a store except Runner.
package reports

import (
	"context"
	"sort"
	"time"

	"shopcore/clock"
	"shopcore/errs"
	"shopcore/ids"
	"shopcore/money"
	"shopcore/orders"
	"shopcore/payments"
	"shopcore/refunds"
)

// DailySales is one row of the daily sales report.
type DailySales struct {
	Day       time.Time
	Orders    int
	Units     int
	Gross     money.Cents
	Discounts money.Cents
	Tax       money.Cents
	Refunded  money.Cents
	Net       money.Cents
}

// BuildDailySales aggregates the orders placed and refunds issued on day
// (UTC). Cancelled orders are excluded.
func BuildDailySales(day time.Time, os []*orders.Order, rs []*refunds.Refund) DailySales {
	start := clock.StartOfDay(day.UTC())
	end := start.Add(24 * time.Hour)
	row := DailySales{Day: start}
	for _, o := range os {
		if o.Status == orders.StatusCancelled || o.PlacedAt.Before(start) || !o.PlacedAt.Before(end) {
			continue
		}
		row.Orders++
		row.Units += o.ItemCount()
		row.Gross += o.Subtotal
		row.Discounts += o.Discount
		row.Tax += o.Tax
	}
	for _, r := range rs {
		if r.Status == refunds.StatusRejected || r.IssuedAt.Before(start) || !r.IssuedAt.Before(end) {
			continue
		}
		row.Refunded += r.Amount
	}
	row.Net = row.Gross - row.Discounts - row.Refunded
	return row
}

// CustomerTotal is a row of the top customers report.
type CustomerTotal struct {
	CustomerID string
	Orders     int
	Spend      money.Cents
}

// TopCustomers ranks customers by spend, highest first, ties by id.
func TopCustomers(os []*orders.Order, n int) []CustomerTotal {
	acc := map[string]*CustomerTotal{}
	for _, o := range os {
		if o.Status == orders.StatusCancelled {
			continue
		}
		ct := acc[o.CustomerID]
		if ct == nil {
			ct = &CustomerTotal{CustomerID: o.CustomerID}
			acc[o.CustomerID] = ct
		}
		ct.Orders++
		ct.Spend += o.Total
	}
	out := make([]CustomerTotal, 0, len(acc))
	for _, ct := range acc {
		out = append(out, *ct)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Spend != out[j].Spend {
			return out[i].Spend > out[j].Spend
		}
		return out[i].CustomerID < out[j].CustomerID
	})
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// CaptureRate is captured / authorized amount over payments, in basis points.
func CaptureRate(ps []*payments.Payment) int {
	var auth, capt money.Cents
	for _, p := range ps {
		auth += p.Amount
		capt += p.Captured
	}
	if auth == 0 {
		return 0
	}
	return int(int64(capt) * 10000 / int64(auth))
}

// Runner executes a report and keeps a ReportRun record of it.
type Runner struct {
	runs  ReportRunStore
	clock clock.Clock
	ids   ids.Generator
}

func NewRunner(runs ReportRunStore, clk clock.Clock, gen ids.Generator) *Runner {
	return &Runner{runs: runs, clock: clock.OrSystem(clk), ids: ids.OrRandom(gen)}
}

// Run executes fn, which returns the number of rows produced.
func (r *Runner) Run(ctx context.Context, kind, requestedBy string, fn func(context.Context) (int, error)) (*ReportRun, error) {
	const op = "reports.run"
	now := r.clock.Now()
	run := &ReportRun{ID: r.ids.Next("rpt"), Kind: kind, RequestedBy: requestedBy, Format: "csv", Status: ReportRunStatusRunning, CreatedAt: now, UpdatedAt: now}
	if err := r.runs.Save(ctx, run); err != nil {
		return nil, errs.Wrap(op, err)
	}
	rows, fnErr := fn(ctx)
	run.Rows = rows
	run.Status = ReportRunStatusDone
	if fnErr != nil {
		run.Status = ReportRunStatusFailed
	}
	run.UpdatedAt = r.clock.Now()
	if err := r.runs.Save(ctx, run); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return run, errs.Wrap(op, fnErr)
}
