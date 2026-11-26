package checkout

import (
	"testing"

	"checkout/promo"
)

func TestQuoteReloadIsStable(t *testing.T) {
	book := promo.Book{
		"SAVE10": {Code: "SAVE10", Percent: 10},
		"OFF3":   {Code: "OFF3", OffCents: 300},
	}
	cart := Cart{SubtotalCents: 1000, Codes: []string{"SAVE10", "OFF3"}}
	for reload := 0; reload < 10; reload++ {
		got, err := Quote(cart, book)
		if err != nil {
			t.Fatal(err)
		}
		if got != 600 {
			t.Fatalf("reload %d: Quote = %d cents, want 600", reload, got)
		}
	}
}
