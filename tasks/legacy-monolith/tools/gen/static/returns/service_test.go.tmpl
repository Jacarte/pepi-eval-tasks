package returns

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/ids"
)

func TestAuthorizeReject(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-rma")
	rec := audit.NewMemory(nil)
	svc := NewService(NewMemoryReturnRequestStore(), rec, nil, ids.NewSequence())
	a, err := svc.Request(ctx, "ord-1", "SKU-1", 1, "too small")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := svc.Request(ctx, "ord-1", "SKU-2", 2, "changed mind")
	rec.Reset()
	if _, err := svc.Authorize(ctx, a.ID, "jo"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reject(ctx, b.ID, "outside window"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Reject(ctx, a.ID, "again"); !errors.Is(err, ErrNotRequested) {
		t.Fatalf("err = %v", err)
	}
	evs := rec.Events()
	if len(evs) != 2 || evs[0].Name != "return_request.authorized" || evs[0].Subject != a.ID || evs[1].Name != "return_request.rejected" {
		t.Fatalf("events = %v", evs)
	}
}
