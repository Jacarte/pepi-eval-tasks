package inventory

import "context"

// Reserve holds qty units of sku for an order.
func (s *Service) Reserve(ctx context.Context, sku string, qty int) (*StockItem, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	it, err := s.repo.Get(ctx, sku)
	if err != nil {
		return nil, err
	}
	if it.Available() < qty {
		return nil, ErrInsufficientStock
	}
	it.Reserved += qty
	it.UpdatedAt = s.clock.Now()
	if err := s.repo.Put(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}
