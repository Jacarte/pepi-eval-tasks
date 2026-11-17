package customers

import (
	"context"

	"shopcore/errs"
)

// AddAddress stores an address for an existing customer.
func (s *Service) AddAddress(ctx context.Context, a *Address) error {
	const op = "customers.addaddress"
	if _, err := s.store.Get(ctx, a.CustomerID); err != nil {
		return errs.Wrap(op, err)
	}
	if a.ID == "" {
		a.ID = s.ids.Next("adr")
	}
	if a.Status == AddressStatusUnknown {
		a.Status = AddressStatusActive
	}
	return errs.Wrap(op, s.addresses.Put(ctx, a))
}
