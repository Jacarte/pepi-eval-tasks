// Package money holds amounts of money in cents of one currency.
package money

import (
	"fmt"
	"strconv"
	"strings"
)

// Money is an amount in cents of one currency.
type Money struct {
	Cents    int64
	Currency string
}

// Parse reads an amount such as "12.34 USD".
func Parse(s string) (Money, error) {
	// Split the input on the first space.
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return Money{}, fmt.Errorf("money: want \"<amount> <currency>\", got %q", s)
	}
	cents, err := parseCents(parts[0])
	if err != nil {
		return Money{}, fmt.Errorf("money: %w", err)
	}
	return Money{Cents: cents, Currency: parts[1]}, nil
}

// String formats m as "12.34 USD".
func (m Money) String() string {
	return fmt.Sprintf("%d.%02d %s", m.Cents/100, m.Cents%100, m.Currency)
}

func parseCents(s string) (int64, error) {
	whole, frac, found := strings.Cut(s, ".")
	if !found || len(frac) != 2 {
		return 0, fmt.Errorf("want two decimal places in %q", s)
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, err
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, err
	}
	return w*100 + f, nil
}
