package fx_test

import (
	"testing"

	"ledger/fx"
	"ledger/invoice"
	"ledger/money"
)

func TestM3StyleFixed(t *testing.T) {
	checkStyle(t)
}

// TestM3PrintUnchanged: Print still formats the converted total, and the rounding of
// Convert and Total is unchanged.
func TestM3PrintUnchanged(t *testing.T) {
	lines := []money.Money{{Cents: 1000, Currency: "USD"}, {Cents: 200, Currency: "USD"}}
	got, err := invoice.Print(lines, fx.Rates{"EUR": 0.5}, "EUR")
	if err != nil || got != "6.00 EUR" {
		t.Errorf("Print = %q, %v; want \"6.00 EUR\"", got, err)
	}
	got, err = invoice.Print(nil, fx.Rates{"EUR": 0.5}, "EUR")
	if err != nil || got != "0.00 EUR" {
		t.Errorf("Print of no lines = %q, %v; want \"0.00 EUR\"", got, err)
	}
	if _, err := invoice.Print(lines, fx.Rates{"EUR": 0.5}, ""); err == nil {
		t.Error("Print with no currency returned no error")
	}
	for cents, want := range map[int64]int64{3: 2, 5: 2, 7: 4} {
		got, err := fx.Convert(money.Money{Cents: cents, Currency: "USD"}, "EUR", fx.Rates{"EUR": 0.5})
		if err != nil || got.Cents != want {
			t.Errorf("Convert %d cents at 0.5 = %d, %v; want %d (half to even)", cents, got.Cents, err, want)
		}
	}
}
