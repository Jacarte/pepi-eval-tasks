package billing

import "testing"

func TestInvoiceTotals(t *testing.T) {
	cases := []struct {
		name string
		inv  Invoice
		want Totals
	}{
		{
			name: "exact tax",
			inv:  Invoice{Lines: []Line{{UnitCents: 1000, Quantity: 2, TaxBP: 1000}}},
			want: Totals{SubtotalCents: 2000, TaxCents: 200, TotalCents: 2200},
		},
		{
			name: "below half rounds down",
			inv:  Invoice{Lines: []Line{{UnitCents: 1001, Quantity: 1, TaxBP: 500}}},
			want: Totals{SubtotalCents: 1001, TaxCents: 50, TotalCents: 1051},
		},
		{
			name: "half rounds up",
			inv:  Invoice{Lines: []Line{{UnitCents: 1050, Quantity: 1, TaxBP: 500}}},
			want: Totals{SubtotalCents: 1050, TaxCents: 53, TotalCents: 1103},
		},
		{
			name: "refund of the same line",
			inv:  Invoice{Lines: []Line{{UnitCents: 1050, Quantity: -1, TaxBP: 500}}},
			want: Totals{SubtotalCents: -1050, TaxCents: -53, TotalCents: -1103},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.inv.Totals(); got != c.want {
				t.Errorf("Totals() = %+v, want %+v", got, c.want)
			}
		})
	}
}
