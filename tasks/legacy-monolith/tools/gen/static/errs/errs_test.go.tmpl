package errs

import (
	"errors"
	"fmt"
	"testing"
)

var errBoom = errors.New("boom")

func TestWrapNil(t *testing.T) {
	if Wrap("x.y", nil) != nil {
		t.Fatal("Wrap(nil) must be nil")
	}
}

func TestWrapChain(t *testing.T) {
	err := Wrap("orders.cancel", fmt.Errorf("load: %w", Wrap("orders.memrepo", errBoom)))
	if !errors.Is(err, errBoom) {
		t.Fatal("lost the cause")
	}
	if Op(err) != "orders.cancel" {
		t.Fatalf("Op = %q", Op(err))
	}
	if got := Trace(err); got != "orders.cancel > orders.memrepo" {
		t.Fatalf("Trace = %q", got)
	}
	if err.Error() != "orders.cancel: load: orders.memrepo: boom" {
		t.Fatalf("Error = %q", err.Error())
	}
}

func TestNew(t *testing.T) {
	err := New("tax.for", "no region")
	if Op(err) != "tax.for" || err.Error() != "tax.for: no region" {
		t.Fatalf("got %q / %q", Op(err), err.Error())
	}
	if Op(errBoom) != "" {
		t.Fatal("plain error has no op")
	}
}
