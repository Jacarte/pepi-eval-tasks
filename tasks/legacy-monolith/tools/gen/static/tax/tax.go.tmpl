// Package tax computes sales tax and VAT from the region table.
package tax

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"shopcore/clock"
	"shopcore/errs"
	"shopcore/money"
)

type regionRate struct {
	Code  string
	Name  string
	BPS   int
	Since string
	Notes string
}

// ErrUnknownRegion is returned for codes missing from the region table.
var ErrUnknownRegion = errors.New("tax: unknown region")

var (
	indexOnce sync.Once
	byCode    map[string]regionRate
)

func index() map[string]regionRate {
	indexOnce.Do(func() {
		byCode = make(map[string]regionRate, len(regionTable))
		for _, r := range regionTable {
			byCode[r.Code] = r
		}
	})
	return byCode
}

// RateBPS returns the rate for region in basis points.
func RateBPS(region string) (int, bool) {
	r, ok := index()[strings.ToUpper(strings.TrimSpace(region))]
	return r.BPS, ok
}

// Compute returns the tax due on amount in region.
func Compute(region string, amount money.Cents) (money.Cents, error) {
	bps, ok := RateBPS(region)
	if !ok {
		return 0, fmt.Errorf("%w: %q", ErrUnknownRegion, region)
	}
	return amount.MulBPS(bps), nil
}

// Regions returns every known region code, sorted.
func Regions() []string {
	out := make([]string, 0, len(regionTable))
	for code := range index() {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// Calculator applies customer exemptions on top of the region table.
type Calculator struct {
	exemptions ExemptionStore
	clock      clock.Clock
}

// NewCalculator returns a calculator reading exemptions from store.
func NewCalculator(store ExemptionStore, clk clock.Clock) *Calculator {
	return &Calculator{exemptions: store, clock: clock.OrSystem(clk)}
}

// For returns the tax customerID owes on amount in region.
func (c *Calculator) For(ctx context.Context, customerID, region string, amount money.Cents) (money.Cents, error) {
	const op = "tax.for"
	list, err := c.exemptions.Scan(ctx, ExemptionFilter{CustomerID: customerID, Status: ExemptionStatusValid})
	if err != nil {
		return 0, errs.Wrap(op, err)
	}
	now := c.clock.Now()
	for _, ex := range list {
		if ex.ExpiresAt.IsZero() || now.Before(ex.ExpiresAt) {
			if ex.Region == "" || strings.EqualFold(ex.Region, region) {
				return 0, nil
			}
		}
	}
	due, err := Compute(region, amount)
	if err != nil {
		return 0, errs.Wrap(op, err)
	}
	return due, nil
}
