package inventory

import (
	"context"
	"errors"
	"testing"
)

func TestReceiveReserveRelease(t *testing.T) {
	ctx := context.Background()
	svc := NewService(NewMemoryRepo())
	if _, err := svc.Receive(ctx, "SKU-1", "ams", 10); err != nil {
		t.Fatal(err)
	}
	it, err := svc.Reserve(ctx, "SKU-1", 4)
	if err != nil || it.Reserved != 4 || it.Available() != 6 {
		t.Fatalf("reserve = %+v, %v", it, err)
	}
	if _, err := svc.Reserve(ctx, "SKU-1", 7); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("err = %v", err)
	}
	it, err = svc.Release(ctx, "SKU-1", 10)
	if err != nil || it.Reserved != 0 {
		t.Fatalf("release = %+v, %v", it, err)
	}
	if _, err := svc.Reserve(ctx, "SKU-404", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Reserve(ctx, "SKU-1", 0); !errors.Is(err, ErrInvalidQuantity) {
		t.Fatalf("err = %v", err)
	}
}
