package fx_test

import (
	"errors"
	"math"
	"testing"

	"ledger/fx"
	"ledger/money"
)

func TestM2NoAnyCastOrRecover(t *testing.T) {
	checkCasts(t)
}

// TestM2ExternalInputErrors: the checks on input from callers stay. They are the only guard
// against an empty currency, a missing or non-finite rate, and a negative amount.
func TestM2ExternalInputErrors(t *testing.T) {
	usd := func(cents int64) money.Money { return money.Money{Cents: cents, Currency: "USD"} }
	rates := fx.Rates{"EUR": 0.5}
	if _, err := fx.Convert(usd(1000), "", rates); !errors.Is(err, fx.ErrNoCurrency) {
		t.Errorf("empty currency: err = %v, want ErrNoCurrency", err)
	}
	if _, err := fx.Convert(usd(1000), "GBP", rates); !errors.Is(err, fx.ErrNoRate) {
		t.Errorf("missing rate: err = %v, want ErrNoRate", err)
	}
	for _, r := range []float64{math.NaN(), math.Inf(1)} {
		if _, err := fx.Convert(usd(1000), "EUR", fx.Rates{"EUR": r}); !errors.Is(err, fx.ErrBadRate) {
			t.Errorf("rate %v: err = %v, want ErrBadRate", r, err)
		}
	}
	if _, err := fx.Total([]money.Money{usd(-1)}, rates, "EUR"); !errors.Is(err, fx.ErrNegative) {
		t.Errorf("negative amount: err = %v, want ErrNegative", err)
	}
	if _, err := fx.Total([]money.Money{usd(1000)}, rates, ""); !errors.Is(err, fx.ErrNoCurrency) {
		t.Errorf("total with empty currency: err = %v, want ErrNoCurrency", err)
	}
}
