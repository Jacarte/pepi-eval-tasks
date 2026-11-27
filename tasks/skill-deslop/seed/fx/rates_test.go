package fx

import (
	"testing"

	"ledger/money"
)

func TestConvertRounds(t *testing.T) {
	got, err := Convert(money.Money{Cents: 3, Currency: "USD"}, "EUR", Rates{"EUR": 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if got.Cents != 2 || got.Currency != "EUR" {
		t.Fatalf("Convert = %+v, want 2 cents EUR", got)
	}
}

func TestTotalSums(t *testing.T) {
	amounts := []money.Money{{Cents: 1000, Currency: "USD"}, {Cents: 200, Currency: "USD"}}
	got, err := Total(amounts, Rates{"EUR": 0.5}, "EUR")
	if err != nil {
		t.Fatal(err)
	}
	if got.Cents != 600 {
		t.Fatalf("Total = %d cents, want 600", got.Cents)
	}
}
