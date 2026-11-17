package catalog

import (
	"context"
	"errors"
	"fmt"

	"shopcore/money"
)

// ErrRetired is returned when a retired listing is changed.
var ErrRetired = errors.New("catalog: listing is retired")

// Reprice changes the list price of a draft or published listing.
func (s *Service) Reprice(ctx context.Context, id string, price money.Cents) (*ProductListing, error) {
	if price <= 0 {
		return nil, fmt.Errorf("reprice: %w", ErrNoPrice)
	}
	l, err := s.listings.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("reprice %s: %w", id, err)
	}
	if l.Status == ProductListingStatusRetired {
		return nil, fmt.Errorf("reprice %s: %w", id, ErrRetired)
	}
	l.Price = price
	l.UpdatedAt = s.clock.Now()
	if err := s.listings.Put(ctx, l); err != nil {
		return nil, fmt.Errorf("reprice %s: %w", id, err)
	}
	return l, nil
}
