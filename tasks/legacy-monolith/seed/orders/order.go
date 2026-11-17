// Package orders owns the order aggregate: placing, cancelling and fulfilling
// orders. Prices come from the pricing package and tax from the tax package.
package orders

import (
	"errors"
	"time"

	"shopcore/money"
)

// Status of an order.
type Status string

const (
	StatusPlaced    Status = "placed"
	StatusCancelled Status = "cancelled"
	StatusFulfilled Status = "fulfilled"
)

var (
	ErrEmptyOrder      = errors.New("orders: order has no lines")
	ErrInvalidQuantity = errors.New("orders: quantity must be positive")
	ErrMissingCustomer = errors.New("orders: customer id is required")
	ErrTooManyLines    = errors.New("orders: too many lines")
	ErrNotCancellable  = errors.New("orders: order can no longer be cancelled")
)

// MaxLines mirrors orders.max_lines in the default config.
const MaxLines = 200

// Line is one SKU on an order.
type Line struct {
	SKU       string
	Qty       int
	UnitPrice money.Cents
}

// Total is Qty * UnitPrice.
func (l Line) Total() money.Cents { return l.UnitPrice * money.Cents(l.Qty) }

// Order is a placed customer order.
type Order struct {
	ID         string
	CustomerID string
	Region     string
	Lines      []Line
	Subtotal   money.Cents
	Discount   money.Cents
	Tax        money.Cents
	Total      money.Cents
	CouponCode string
	Status     Status
	PlacedAt   time.Time
	UpdatedAt  time.Time
}

// Clone returns a deep copy.
func (o *Order) Clone() *Order {
	if o == nil {
		return nil
	}
	cp := *o
	cp.Lines = append([]Line(nil), o.Lines...)
	return &cp
}

// ItemCount is the total number of units across all lines.
func (o *Order) ItemCount() int {
	n := 0
	for _, l := range o.Lines {
		n += l.Qty
	}
	return n
}
