package returns

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleReturnRequest(n int) *ReturnRequest {
	return &ReturnRequest{
		ID:        fmt.Sprintf("return_request-%03d", n),
		OrderID:   fmt.Sprintf("orderid %d;a=b,c\\d", n),
		SKU:       fmt.Sprintf("sku-%d", n),
		Qty:       n*36 + 98,
		Reason:    fmt.Sprintf("reason-%d", n),
		Inspector: fmt.Sprintf("inspector %d;a=b,c\\d", n),
		Status:    ReturnRequestStatusRequested,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestReturnRequestCodecRoundTrip(t *testing.T) {
	in := sampleReturnRequest(4)
	line := EncodeReturnRequest(in)
	out, err := DecodeReturnRequest(line)
	if err != nil {
		t.Fatalf("DecodeReturnRequest(%q): %v", line, err)
	}
	if got := EncodeReturnRequest(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.OrderID != in.OrderID {
		t.Errorf("OrderID = %q, want %q", out.OrderID, in.OrderID)
	}
	if out.SKU != in.SKU {
		t.Errorf("SKU = %q, want %q", out.SKU, in.SKU)
	}
	if out.Reason != in.Reason {
		t.Errorf("Reason = %q, want %q", out.Reason, in.Reason)
	}
	if out.Inspector != in.Inspector {
		t.Errorf("Inspector = %q, want %q", out.Inspector, in.Inspector)
	}
	if _, err := DecodeReturnRequest("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestReturnRequestStatusTransitions(t *testing.T) {
	for _, s := range []ReturnRequestStatus{ReturnRequestStatusRequested, ReturnRequestStatusRejected} {
		got, err := ParseReturnRequestStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ReturnRequestStatusRequested.CanTransition(ReturnRequestStatusAuthorized) {
		t.Fatal("first transition should be allowed")
	}
	if ReturnRequestStatusAuthorized.CanTransition(ReturnRequestStatusRequested) {
		t.Fatal("backwards transition allowed")
	}
	if !ReturnRequestStatusRejected.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
