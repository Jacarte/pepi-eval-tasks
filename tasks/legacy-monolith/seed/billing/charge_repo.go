package billing

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ChargeStore persists charges.
type ChargeStore interface {
	Get(ctx context.Context, id string) (*Charge, error)
	Put(ctx context.Context, v *Charge) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f ChargeFilter) ([]*Charge, error)
}

// ChargeFilter narrows List results. Zero fields match everything.
type ChargeFilter struct {
	Status    ChargeStatus
	InvoiceID string
	Limit     int
}

func (f ChargeFilter) match(v *Charge) bool {
	if f.Status != ChargeStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.InvoiceID != "" && v.InvoiceID != f.InvoiceID {
		return false
	}
	return true
}

// MemoryChargeStore keeps charges in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryChargeStore struct {
	mu    sync.RWMutex
	items map[string]*Charge
}

// NewMemoryChargeStore returns an empty store.
func NewMemoryChargeStore() *MemoryChargeStore {
	return &MemoryChargeStore{items: make(map[string]*Charge)}
}

var _ ChargeStore = (*MemoryChargeStore)(nil)

func (s *MemoryChargeStore) Get(ctx context.Context, id string) (*Charge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: charge %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemoryChargeStore) Put(ctx context.Context, v *Charge) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil charge", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemoryChargeStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: charge %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// List returns matching charges ordered by id.
func (s *MemoryChargeStore) List(ctx context.Context, f ChargeFilter) ([]*Charge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*Charge, 0, len(s.items))
	for _, v := range s.items {
		if f.match(v) {
			out = append(out, v.Clone())
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// Len reports how many charges are stored.
func (s *MemoryChargeStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
