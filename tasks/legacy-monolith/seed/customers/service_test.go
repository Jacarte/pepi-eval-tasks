package customers

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

func TestRegisterSuspendClose(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-c")
	rec := audit.NewMemory(nil)
	svc := NewService(NewMemoryCustomerStore(), NewMemoryAddressStore(), rec, nil, ids.NewSequence())
	c, err := svc.Register(ctx, "Ada Lovelace <ADA@example.com>", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Email != "ada@example.com" || c.DisplayName != "Ada Lovelace" {
		t.Fatalf("customer = %+v", c)
	}
	if _, err := svc.Register(ctx, "ada@example.com", "dup"); !errors.Is(err, ErrDuplicated) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Suspend(ctx, c.ID, "chargebacks"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Close(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Close(ctx, c.ID); !errors.Is(err, ErrClosed) || errs.Op(err) != "customers.close" {
		t.Fatalf("err = %v", err)
	}
	var names []string
	for _, ev := range rec.ForSubject(c.ID) {
		names = append(names, ev.Name)
	}
	if len(names) != 3 || names[0] != "customer.registered" || names[2] != "customer.closed" {
		t.Fatalf("events = %v", names)
	}
}

func TestAddAddress(t *testing.T) {
	ctx := context.Background()
	addrs := NewMemoryAddressStore()
	svc := NewService(NewMemoryCustomerStore(), addrs, nil, nil, ids.NewSequence())
	c, _ := svc.Register(ctx, "bob@example.com", "Bob")
	if err := svc.AddAddress(ctx, &Address{CustomerID: c.ID, Line1: "1 Main St", Country: "US"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddAddress(ctx, &Address{CustomerID: "cus-404", Line1: "x", Country: "US"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if addrs.Len() != 1 {
		t.Fatalf("addresses = %d", addrs.Len())
	}
}
