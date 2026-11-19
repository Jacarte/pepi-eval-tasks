package billing

import "testing"

// wantTax is the rule of the README: round to the nearest cent, halves away from zero.
func wantTax(amount, bp int64) int64 {
	n := amount * bp
	if n < 0 {
		return -((-n + 5000) / 10000)
	}
	return (n + 5000) / 10000
}

func TestM1LineTax(t *testing.T) {
	cases := []struct {
		amount, bp, want int64
	}{
		{1050, 500, 53},
		{-1050, 500, -53},
		{10, 500, 1},
		{-10, 500, -1},
		{1, 4999, 0},
		{1, 5000, 1},
		{-1, 5000, -1},
		{-1, 4999, 0},
		{1999, 825, 165},
		{-1999, 825, -165},
		{0, 825, 0},
		{12345, 0, 0},
	}
	for _, c := range cases {
		l := Line{UnitCents: c.amount, Quantity: 1, TaxBP: c.bp}
		if got := l.TaxCents(); got != c.want {
			t.Errorf("tax of %d cents at %d bp = %d, want %d", c.amount, c.bp, got, c.want)
		}
	}
}

func TestM1LineTaxSweep(t *testing.T) {
	for _, bp := range []int64{50, 100, 500, 625, 825, 1000, 1900} {
		for amount := int64(-3000); amount <= 3000; amount++ {
			l := Line{UnitCents: amount, Quantity: 1, TaxBP: bp}
			if got, want := l.TaxCents(), wantTax(amount, bp); got != want {
				t.Fatalf("tax of %d cents at %d bp = %d, want %d", amount, bp, got, want)
			}
		}
	}
}

func TestM1RefundMirrorsSale(t *testing.T) {
	sale := Line{UnitCents: 1999, Quantity: 3, TaxBP: 825}
	refund := Line{UnitCents: 1999, Quantity: -3, TaxBP: 825}
	if s, r := sale.TaxCents(), refund.TaxCents(); s != -r {
		t.Fatalf("sale tax %d, refund tax %d: want opposite values", s, r)
	}
}

func TestM1InvoiceTotals(t *testing.T) {
	inv := Invoice{ID: "H1", Lines: []Line{
		{UnitCents: 1050, Quantity: 1, TaxBP: 500},
		{UnitCents: 1999, Quantity: 3, TaxBP: 825},
		{UnitCents: 1999, Quantity: -1, TaxBP: 825},
		{UnitCents: 250, Quantity: 2, TaxBP: 0},
	}}
	// Line taxes: 53 + 495 (494.7525 rounds up) - 165 (164.9175 rounds up in magnitude) + 0.
	want := Totals{SubtotalCents: 1050 + 5997 - 1999 + 500, TaxCents: 53 + 495 - 165, TotalCents: 1050 + 5997 - 1999 + 500 + 53 + 495 - 165}
	if got := inv.Totals(); got != want {
		t.Fatalf("Totals() = %+v, want %+v", got, want)
	}
}
