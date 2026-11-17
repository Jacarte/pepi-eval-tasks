// Package customers is the customer master: registration, status changes and
// addresses.
package customers

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
	"shopcore/ids"
)

var (
	ErrBadEmail   = errors.New("customers: invalid email address")
	ErrNotActive  = errors.New("customers: customer is not active")
	ErrClosed     = errors.New("customers: customer account is closed")
	ErrDuplicated = errors.New("customers: email already registered")
)

type Service struct {
	store     CustomerStore
	addresses AddressStore
	audit     audit.Recorder
	clock     clock.Clock
	ids       ids.Generator
}

func NewService(store CustomerStore, addresses AddressStore, rec audit.Recorder, clk clock.Clock, gen ids.Generator) *Service {
	return &Service{store: store, addresses: addresses, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk), ids: ids.OrRandom(gen)}
}

// Register creates an active customer.
func (s *Service) Register(ctx context.Context, email, displayName string) (*Customer, error) {
	const op = "customers.register"
	addr, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return nil, errs.Wrap(op, ErrBadEmail)
	}
	email = strings.ToLower(addr.Address)
	existing, err := s.store.List(ctx, CustomerFilter{Email: email, Limit: 1})
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if len(existing) > 0 {
		return nil, errs.Wrap(op, ErrDuplicated)
	}
	now := s.clock.Now()
	c := &Customer{
		ID:          s.ids.Next("cus"),
		Email:       email,
		DisplayName: firstNonBlank(displayName, addr.Name, email),
		Segment:     "new",
		Status:      CustomerStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Put(ctx, c); err != nil {
		return nil, errs.Wrap(op, err)
	}
	ev := audit.Event{Name: "customer.registered", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: c.ID}
	if err := s.audit.Record(ctx, ev); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return c, nil
}

// Suspend blocks a customer from checking out.
func (s *Service) Suspend(ctx context.Context, id, reason string) (*Customer, error) {
	const op = "customers.suspend"
	c, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if c.Status != CustomerStatusActive {
		return nil, errs.Wrap(op, ErrNotActive)
	}
	c.Status = CustomerStatusSuspended
	c.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, c); err != nil {
		return nil, errs.Wrap(op, err)
	}
	ev := audit.Event{Name: "customer.suspended", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: c.ID, Attrs: map[string]string{"reason": reason}}
	if err := s.audit.Record(ctx, ev); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return c, nil
}

// Close permanently closes an account (GDPR erasure runs separately).
func (s *Service) Close(ctx context.Context, id string) (*Customer, error) {
	const op = "customers.close"
	c, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if c.Status == CustomerStatusClosed {
		return nil, errs.Wrap(op, ErrClosed)
	}
	c.Status = CustomerStatusClosed
	c.Marketing = false
	c.UpdatedAt = s.clock.Now()
	if err := s.store.Put(ctx, c); err != nil {
		return nil, errs.Wrap(op, err)
	}
	ev := audit.Event{Name: "customer.closed", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: c.ID}
	if err := s.audit.Record(ctx, ev); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return c, nil
}

func firstNonBlank(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
