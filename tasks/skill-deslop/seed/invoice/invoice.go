// Package invoice prints invoice totals.
package invoice

import (
	"fmt"

	"ledger/fx"
	"ledger/money"
)

// ParseLegacy reads an amount from the old CSV importer, which writes strings or floats.
func ParseLegacy(v any) (money.Money, error) {
	// The importer sends either a string or a float64.
	switch x := v.(type) {
	case string:
		return money.Parse(x)
	case float64:
		return money.Money{Cents: int64(x * 100), Currency: "USD"}, nil
	}
	return money.Money{}, fmt.Errorf("invoice: unsupported legacy value %T", v)
}

// Print formats the total of amounts in currency, converted with rates.
func Print(amounts []money.Money, rates fx.Rates, currency string) (string, error) {
	// Get the total in the target currency first, then format it.
	total, err := fx.Total(amounts, rates, currency)
	if err != nil {
		return "", fmt.Errorf("invoice: %w", err)
	}
	return total.String(), nil
}
