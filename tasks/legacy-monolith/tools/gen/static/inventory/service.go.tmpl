package inventory

import (
	"context"
	"errors"

	"shopcore/audit"
	"shopcore/clock"
)

// Service manages stock positions.
type Service struct {
	repo  Repository
	audit audit.Recorder
	clock clock.Clock
}

// Option configures a Service.
type Option func(*Service)

// WithAudit sets the audit recorder (default: discard).
func WithAudit(r audit.Recorder) Option { return func(s *Service) { s.audit = audit.OrDiscard(r) } }

// WithClock sets the clock (default: system).
func WithClock(c clock.Clock) Option { return func(s *Service) { s.clock = clock.OrSystem(c) } }

func NewService(repo Repository, opts ...Option) *Service {
	s := &Service{repo: repo, audit: audit.Discard{}, clock: clock.System{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Receive adds qty units of sku to warehouse stock, creating the item if
// needed.
func (s *Service) Receive(ctx context.Context, sku, warehouse string, qty int) (*StockItem, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	it, err := s.repo.Get(ctx, sku)
	if errors.Is(err, ErrNotFound) {
		it, err = &StockItem{ID: sku, Warehouse: warehouse}, nil
	}
	if err != nil {
		return nil, err
	}
	it.OnHand += qty
	it.UpdatedAt = s.clock.Now()
	if err := s.repo.Put(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

// Release returns previously reserved units to the available pool.
func (s *Service) Release(ctx context.Context, sku string, qty int) (*StockItem, error) {
	if qty <= 0 {
		return nil, ErrInvalidQuantity
	}
	it, err := s.repo.Get(ctx, sku)
	if err != nil {
		return nil, err
	}
	if qty > it.Reserved {
		qty = it.Reserved // over-release happens when carts time out twice
	}
	it.Reserved -= qty
	it.UpdatedAt = s.clock.Now()
	if err := s.repo.Put(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

// Snapshot lists every stock item.
func (s *Service) Snapshot(ctx context.Context) ([]*StockItem, error) { return s.repo.All(ctx) }
