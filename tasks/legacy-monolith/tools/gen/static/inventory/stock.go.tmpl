// Package inventory tracks on-hand and reserved stock per SKU.
package inventory

import (
	"errors"
	"time"
)

var (
	ErrInvalidQuantity   = errors.New("inventory: quantity must be positive")
	ErrInsufficientStock = errors.New("inventory: insufficient stock")
)

// StockItem is the stock position of one SKU. ID is the SKU itself.
type StockItem struct {
	ID        string
	Warehouse string
	OnHand    int
	Reserved  int
	UpdatedAt time.Time
}

// Available is what can still be reserved.
func (s *StockItem) Available() int { return s.OnHand - s.Reserved }

func (s *StockItem) clone() *StockItem {
	cp := *s
	return &cp
}
