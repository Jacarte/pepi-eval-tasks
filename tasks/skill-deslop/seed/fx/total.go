package fx

import (
	"errors"
	"fmt"

	"ledger/money"
)

// Total returns the sum of amounts in currency, each converted with rates.
func Total(amounts []money.Money, rates Rates, currency string) (result money.Money, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("fx: total: %v", r)
		}
	}()
	var total int64
	// Loop over the amounts and convert each one.
	for _, amount := range amounts {
		if amount.Cents < 0 {
			return money.Money{}, ErrNegative
		}
		converted, err := Convert(amount, currency, rates)
		if err != nil {
			return money.Money{}, fmt.Errorf("convert line: %w", err)
		}
		if converted.Currency != currency {
			return money.Money{}, errors.New("fx: currency mismatch")
		}
		total += converted.Cents
	}
	return money.Money{Cents: any(total).(int64), Currency: currency}, nil
}
