package refunds

import (
	"context"
	"errors"
	"testing"

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
