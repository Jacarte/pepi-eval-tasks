// Package pricing turns list prices into sell prices by applying rules.
package pricing

import (
	"context"
	"errors"
	"sort"

	"shopcore/errs"
	"shopcore/money"
)

// Rule kinds.
const (
	KindPercent = "percent"
	KindFixed   = "fixed"
)

var ErrNegativeBase = errors.New("pricing: negative base price")

// Quote is the outcome of pricing one SKU.
type Quote struct {
	SKU      string
	Qty      int
	Base     money.Cents
	Discount money.Cents
	Unit     money.Cents
	Total    money.Cents
	Applied  []string // rule ids, in application order
}

// Engine prices SKUs using the active rules in its store.
type Engine struct {
	rules PriceRuleStore
}

func NewEngine(rules PriceRuleStore) *Engine { return &Engine{rules: rules} }

// Quote prices qty units of sku listed at base.
func (e *Engine) Quote(ctx context.Context, sku string, base money.Cents, qty int) (Quote, error) {
	const op = "pricing.quote"
	if base < 0 {
		return Quote{}, errs.Wrap(op, ErrNegativeBase)
	}
	if qty <= 0 {
		qty = 1
	}
	all, err := e.rules.List(ctx, PriceRuleFilter{Status: PriceRuleStatusActive})
	if err != nil {
		return Quote{}, errs.Wrap(op, err)
	}
	var rules []*PriceRule
	for _, r := range all {
		if r.SKU == sku || r.SKU == "*" {
			rules = append(rules, r)
		}
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority < rules[j].Priority })
	q := Quote{SKU: sku, Qty: qty, Base: base}
	for _, r := range rules {
		var d money.Cents
		switch r.Kind {
		case KindPercent:
			d = (base - q.Discount).MulBPS(int(r.Value))
		case KindFixed:
			d = money.Cents(r.Value)
		default:
			continue // unknown kinds come from the old admin UI; ignore them
		}
		q.Discount += d
		q.Applied = append(q.Applied, r.ID)
		if !r.Stackable {
			break
		}
	}
	q.Discount = money.Min(q.Discount, base)
	q.Unit = base - q.Discount
	q.Total = q.Unit * money.Cents(qty)
	return q, nil
}

// ApplyCoupon takes percentOff off price. Kept here rather than in coupons so
// the checkout path has no dependency on the coupon store.
func ApplyCoupon(price money.Cents, percentOff int) money.Cents {
	if percentOff <= 0 {
		return price
	}
	if percentOff >= 100 {
		return 0
	}
	return price - price.Percent(percentOff)
}
