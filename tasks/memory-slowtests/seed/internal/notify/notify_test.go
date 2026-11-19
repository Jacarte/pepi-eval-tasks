package notify

import (
	"testing"

	"shopbill/internal/billing"
)

func TestCents(t *testing.T) {
	for c, want := range map[int64]string{0: "$0.00", 5: "$0.05", 1103: "$11.03", -1103: "-$11.03"} {
		if got := Cents(c); got != want {
			t.Errorf("Cents(%d) = %q, want %q", c, got, want)
		}
	}
}

func TestFormat(t *testing.T) {
	inv := billing.Invoice{ID: "N1", Lines: []billing.Line{{UnitCents: 1000, Quantity: 1, TaxBP: 1000}}}
	if got, want := Format(inv), "Invoice N1: $11.00 (tax $1.00)"; got != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}
