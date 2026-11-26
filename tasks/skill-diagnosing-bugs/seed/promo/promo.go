// Package promo resolves the promotion codes a customer entered against the promotion book.
package promo

import (
	"fmt"

	"checkout/pricing"
)

// Book maps each promotion code to its promotion.
type Book map[string]pricing.Promotion

// Resolve returns the promotions for codes, one per distinct code.
func Resolve(codes []string, book Book) ([]pricing.Promotion, error) {
	chosen := make(map[string]pricing.Promotion, len(codes))
	for _, code := range codes {
		p, ok := book[code]
		if !ok {
			return nil, fmt.Errorf("unknown promotion code %q", code)
		}
		chosen[code] = p
	}
	out := make([]pricing.Promotion, 0, len(chosen))
	for _, p := range chosen {
		out = append(out, p)
	}
	return out, nil
}
