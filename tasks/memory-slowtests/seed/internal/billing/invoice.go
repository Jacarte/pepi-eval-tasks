// Package billing computes invoice lines, tax and totals in integer cents.
package billing

// Line is one invoice line. A negative Quantity is a refund. TaxBP is the tax rate in basis points.
type Line struct {
	Description string
	UnitCents   int64
	Quantity    int64
	TaxBP       int64
}

// Invoice is a list of lines.
type Invoice struct {
	ID    string
	Lines []Line
}

// Totals are the sums over an invoice.
type Totals struct {
	SubtotalCents int64
	TaxCents      int64
	TotalCents    int64
}

// AmountCents is the line amount before tax.
func (l Line) AmountCents() int64 {
	return l.UnitCents * l.Quantity
}

// TaxCents is the tax of the line, rounded to the nearest cent.
func (l Line) TaxCents() int64 {
	return divRound(l.AmountCents()*l.TaxBP, 10000)
}

// Totals sums the amounts and the per-line taxes.
func (inv Invoice) Totals() Totals {
	var t Totals
	for _, l := range inv.Lines {
		t.SubtotalCents += l.AmountCents()
		t.TaxCents += l.TaxCents()
	}
	t.TotalCents = t.SubtotalCents + t.TaxCents
	return t
}
