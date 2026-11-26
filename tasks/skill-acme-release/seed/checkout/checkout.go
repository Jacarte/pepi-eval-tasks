// Package checkout quotes the total of a cart.
package checkout

// Quote returns the subtotal minus the discount, never below zero, in cents.
func Quote(subtotalCents, discountCents int64) int64 {
	if discountCents >= subtotalCents {
		return 0
	}
	return subtotalCents - discountCents
}
