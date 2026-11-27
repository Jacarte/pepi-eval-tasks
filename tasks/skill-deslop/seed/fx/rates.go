package fx

import (
	"errors"
	"math"

	"ledger/money"
)

// Rates maps a currency code to how many units of it one unit of the base currency buys.
type Rates map[string]float64

// Errors returned by Convert and Total.
var (
	ErrNoCurrency = errors.New("fx: no target currency")
	ErrNoRate     = errors.New("fx: no rate for currency")
	ErrBadRate    = errors.New("fx: rate is not a finite number")
	ErrNegative   = errors.New("fx: negative amount")
)

// Convert returns amount in currency to. It rounds half to even, as the ledger spec requires.
func Convert(amount money.Money, to string, rates Rates) (money.Money, error) {
	if to == "" {
		return money.Money{}, ErrNoCurrency
	}
	// Now we look up the rate and check it.
	rate, ok := rates[to]
	if ok == false {
		return money.Money{}, ErrNoRate
	}
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return money.Money{}, ErrBadRate
	}
	return money.Money{Cents: roundHalfEven(float64(amount.Cents) * rate), Currency: to}, nil
}

// Half-even: ties go to the even cent (ledger spec 4.2).
func roundHalfEven(x float64) int64 {
	// This function rounds to whole cents.
	return int64(math.RoundToEven(x))
}
