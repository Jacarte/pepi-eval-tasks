// Package money implements integer minor-unit amounts. Floats are not allowed
// anywhere near prices; see the 2019 rounding incident write-up.
package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Cents is an amount in minor units of the order currency.
type Cents int64

var ErrSyntax = errors.New("money: invalid amount")

// String renders c as "12.34" (or "-0.50").
func (c Cents) String() string {
	sign := ""
	v := int64(c)
	if v < 0 {
		sign = "-"
		v = -v
	}
	return fmt.Sprintf("%s%d.%02d", sign, v/100, v%100)
}

// Parse reads "12", "12.3" or "12.34". More than two decimals is an error.
func Parse(s string) (Cents, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrSyntax
	}
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	whole, frac, _ := strings.Cut(s, ".")
	if len(frac) > 2 || whole == "" {
		return 0, fmt.Errorf("%w: %q", ErrSyntax, s)
	}
	for len(frac) < 2 {
		frac += "0"
	}
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrSyntax, s)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrSyntax, s)
	}
	v := Cents(w*100 + f)
	if neg {
		v = -v
	}
	return v, nil
}

// MulBPS multiplies c by bps/10000, rounding half away from zero.
func (c Cents) MulBPS(bps int) Cents {
	p := int64(c) * int64(bps)
	if p >= 0 {
		return Cents((p + 5000) / 10000)
	}
	return Cents((p - 5000) / 10000)
}

// Percent returns pct percent of c, rounded like MulBPS.
func (c Cents) Percent(pct int) Cents { return c.MulBPS(pct * 100) }

// Sum adds amounts.
func Sum(vals ...Cents) Cents {
	var t Cents
	for _, v := range vals {
		t += v
	}
	return t
}

// Min returns the smaller amount.
func Min(a, b Cents) Cents {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger amount.
func Max(a, b Cents) Cents {
	if a > b {
		return a
	}
	return b
}

// Allocate splits total into n parts that differ by at most one cent. The
// first parts receive the remainder, matching what finance does by hand.
func Allocate(total Cents, n int) []Cents {
	if n <= 0 {
		return nil
	}
	out := make([]Cents, n)
	q, r := int64(total)/int64(n), int64(total)%int64(n)
	for i := range out {
		out[i] = Cents(q)
		if int64(i) < r {
			out[i]++
		}
	}
	return out
}
