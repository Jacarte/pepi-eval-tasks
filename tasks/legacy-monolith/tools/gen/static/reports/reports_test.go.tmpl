package reports

import (
	"context"
	"errors"
	"testing"
	"time"

	"shopcore/ids"
	"shopcore/orders"
	"shopcore/payments"
	"shopcore/refunds"
)

func TestBuildDailySales(t *testing.T) {
	day := time.Date(2024, 3, 5, 15, 0, 0, 0, time.UTC)
	os := []*orders.Order{
		{ID: "o1", CustomerID: "a", Subtotal: 1000, Discount: 100, Tax: 50, Total: 950, Status: orders.StatusPlaced, PlacedAt: day, Lines: []orders.Line{{SKU: "x", Qty: 2}}},
		{ID: "o2", CustomerID: "b", Subtotal: 500, Total: 500, Status: orders.StatusCancelled, PlacedAt: day},
		{ID: "o3", CustomerID: "a", Subtotal: 700, Total: 700, Status: orders.StatusPlaced, PlacedAt: day.Add(24 * time.Hour)},
	}
	rs := []*refunds.Refund{{ID: "r1", Amount: 200, Status: refunds.StatusIssued, IssuedAt: day}}
	row := BuildDailySales(day, os, rs)
	if row.Orders != 1 || row.Units != 2 || row.Gross != 1000 || row.Refunded != 200 || row.Net != 700 {
		t.Fatalf("row = %+v", row)
	}
	top := TopCustomers(os, 1)
	if len(top) != 1 || top[0].CustomerID != "a" || top[0].Spend != 1650 {
		t.Fatalf("top = %+v", top)
	}
}

func TestCaptureRate(t *testing.T) {
	ps := []*payments.Payment{{Amount: 1000, Captured: 1000}, {Amount: 1000, Captured: 500}}
	if got := CaptureRate(ps); got != 7500 {
		t.Fatalf("rate = %d", got)
	}
	if CaptureRate(nil) != 0 {
		t.Fatal("empty rate")
	}
}

func TestRunnerRecordsFailures(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryReportRunStore()
	r := NewRunner(store, nil, ids.NewSequence())
	run, err := r.Run(ctx, "daily_sales", "cron", func(context.Context) (int, error) { return 12, nil })
	if err != nil || run.Status != ReportRunStatusDone || run.Rows != 12 {
		t.Fatalf("run = %+v, %v", run, err)
	}
	boom := errors.New("boom")
	run, err = r.Run(ctx, "daily_sales", "cron", func(context.Context) (int, error) { return 0, boom })
	if !errors.Is(err, boom) || run.Status != ReportRunStatusFailed {
		t.Fatalf("run = %+v, %v", run, err)
	}
}
