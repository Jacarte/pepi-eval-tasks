package checkout_test

import (
	"testing"

	"checkout/checkout"
	"checkout/promo"
)

// Each case is quoted many times: a fix must hold on every call, not only on a lucky one.
const repeats = 100

var m1Book = promo.Book{
	"SAVE10": {Code: "SAVE10", Percent: 10},
	"VIP25":  {Code: "VIP25", Percent: 25},
	"OFF3":   {Code: "OFF3", OffCents: 300},
	"OFF5":   {Code: "OFF5", OffCents: 500},
}

func TestM1QuoteEnteredOrder(t *testing.T) {
	cases := []struct {
		name     string
		subtotal int64
		codes    []string
		want     int64
	}{
		{"percent then fixed", 1000, []string{"SAVE10", "OFF3"}, 600},
		{"fixed then percent", 1000, []string{"OFF3", "SAVE10"}, 630},
		{"big percent first", 2500, []string{"VIP25", "OFF5"}, 1375},
		{"big fixed first", 2500, []string{"OFF5", "VIP25"}, 1500},
		{"three codes", 2000, []string{"SAVE10", "VIP25", "OFF3"}, 1050},
		{"three codes reversed", 2000, []string{"OFF3", "VIP25", "SAVE10"}, 1148},
		{"percent rounds down", 999, []string{"SAVE10"}, 900},
		{"fixed never below zero", 700, []string{"OFF3", "OFF5"}, 0},
		{"no codes", 1234, nil, 1234},
	}
	for _, c := range cases {
		for i := 0; i < repeats; i++ {
			got, err := checkout.Quote(checkout.Cart{SubtotalCents: c.subtotal, Codes: c.codes}, m1Book)
			if err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
			if got != c.want {
				t.Fatalf("%s: call %d: Quote = %d cents, want %d", c.name, i, got, c.want)
			}
		}
	}
}
