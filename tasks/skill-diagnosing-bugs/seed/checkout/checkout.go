// Package checkout quotes the total of a cart after its promotion codes.
package checkout

import (
	"checkout/pricing"
	"checkout/promo"
)

// Cart is what the customer is paying for.
type Cart struct {
	SubtotalCents int64
	// Codes are the promotion codes in the order the customer entered them.
	Codes []string
}

// Quote returns the total in cents of cart after its promotions.
func Quote(cart Cart, book promo.Book) (int64, error) {
	promos, err := promo.Resolve(cart.Codes, book)
	if err != nil {
		return 0, err
	}
	return pricing.Apply(cart.SubtotalCents, promos), nil
}
