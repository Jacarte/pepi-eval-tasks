// Package pricing applies promotions to a cart subtotal, in cents.
package pricing

// Promotion is one code: a percentage (Percent 1 to 100) or a fixed amount (OffCents above 0).
type Promotion struct {
	Code     string
	Percent  int64
	OffCents int64
}

// Apply returns the total in cents after promos, applied in the order given.
func Apply(subtotalCents int64, promos []Promotion) int64 {
	total := subtotalCents
	for _, p := range promos {
		if p.Percent > 0 {
			total -= total * p.Percent / 100
		} else {
			total = max(0, total-p.OffCents)
		}
	}
	return total
}
