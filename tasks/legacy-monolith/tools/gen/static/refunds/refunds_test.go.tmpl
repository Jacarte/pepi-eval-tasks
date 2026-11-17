package refunds

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

func TestIssueAndSettle(t *testing.T) {
	ctx := context.Background()
	svc := New(Deps{Repo: NewMemoryRepo(), IDs: ids.NewSequence(), MaxAmount: 10000})
	r, err := svc.Issue(ctx, IssueRequest{PaymentID: "pay-1", OrderID: "ord-1", Amount: 1500, Reason: ReasonDamaged})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "rfd-000001" || r.Status != StatusIssued {
		t.Fatalf("refund = %+v", r)
	}
	if _, err := svc.Settle(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Settle(ctx, r.ID); !errors.Is(err, ErrNotIssued) {
		t.Fatalf("err = %v", err)
	}
	total, err := svc.TotalForPayment(ctx, "pay-1")
	if err != nil || total != 1500 {
		t.Fatalf("total = %v, %v", total, err)
	}
}

func TestIssueValidation(t *testing.T) {
	ctx := context.Background()
	svc := New(Deps{Repo: NewMemoryRepo(), IDs: ids.NewSequence(), MaxAmount: 1000})
	for _, tc := range []struct {
		req  IssueRequest
		want error
	}{
		{IssueRequest{Amount: 10, Reason: ReasonGoodwill}, ErrMissingPayment},
		{IssueRequest{PaymentID: "p", Amount: 0, Reason: ReasonGoodwill}, ErrInvalidAmount},
		{IssueRequest{PaymentID: "p", Amount: 10, Reason: "because"}, ErrUnknownReason},
		{IssueRequest{PaymentID: "p", Amount: 1001, Reason: ReasonGoodwill}, ErrOverLimit},
	} {
		if _, err := svc.Issue(ctx, tc.req); !errors.Is(err, tc.want) {
			t.Errorf("Issue(%+v) err = %v, want %v", tc.req, err, tc.want)
		}
	}
}

func TestSettleIsAudited(t *testing.T) {
	rec := audit.NewMemory(nil)
	repo := NewMemoryRepo()
	svc := New(Deps{Repo: repo, Audit: rec, IDs: ids.NewSequence()})
	ctx := ctxkeys.WithRequestID(context.Background(), "req-settle")
	_ = repo.Insert(ctx, &Refund{ID: "rfd-9", PaymentID: "pay-3", OrderID: "ord-3", Amount: 700, Reason: ReasonGoodwill, Status: StatusIssued})
	if _, err := svc.Settle(ctx, "rfd-9"); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	if len(evs) != 1 || evs[0].Name != "refund.settled" || evs[0].Subject != "rfd-9" || evs[0].RequestID != "req-settle" {
		t.Fatalf("events = %v", evs)
	}
	if evs[0].Attrs["payment_id"] != "pay-3" || evs[0].Attrs["reason"] != "goodwill" {
		t.Fatalf("attrs = %v", evs[0].Attrs)
	}
	if _, err := svc.Settle(ctx, "rfd-9"); errs.Op(err) != "refunds.settle" {
		t.Fatalf("op = %q", errs.Op(err))
	}
}
