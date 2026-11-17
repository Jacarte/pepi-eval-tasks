package tax

import (
	"context"
	"errors"
	"testing"
	"time"

	"shopcore/clock"
	"shopcore/errs"
)

func TestCompute(t *testing.T) {
	got, err := Compute("us-ca", 10000)
	if err != nil || got != 725 {
		t.Fatalf("Compute = %v, %v", got, err)
	}
	if _, err := Compute("XX", 100); !errors.Is(err, ErrUnknownRegion) {
		t.Fatalf("err = %v", err)
	}
	if len(Regions()) != len(regionTable) {
		t.Fatal("duplicate region codes in table")
	}
}

func TestCalculatorExemption(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryExemptionStore()
	clk := clock.NewFixed(time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC))
	ex := &Exemption{ID: "ex-1", CustomerID: "c-1", Certificate: "CERT-9", Region: "US-NY", Status: ExemptionStatusValid, ExpiresAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)}
	if err := store.Store(ctx, ex); err != nil {
		t.Fatal(err)
	}
	calc := NewCalculator(store, clk)
	if due, err := calc.For(ctx, "c-1", "US-NY", 10000); err != nil || due != 0 {
		t.Fatalf("exempt: %v %v", due, err)
	}
	if due, _ := calc.For(ctx, "c-1", "US-CA", 10000); due != 725 {
		t.Fatalf("other region: %v", due)
	}
	clk.Set(time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC))
	if due, _ := calc.For(ctx, "c-1", "US-NY", 10000); due != 400 {
		t.Fatalf("expired exemption still applied: %v", due)
	}
	if _, err := calc.For(ctx, "c-2", "ZZ", 1); errs.Op(err) != "tax.for" {
		t.Fatalf("err = %v", err)
	}
}
