package suppliers

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// SupplierStore persists suppliers.
type SupplierStore interface {
	Find(ctx context.Context, id string) (*Supplier, error)
	Save(ctx context.Context, v *Supplier) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f SupplierFilter) ([]*Supplier, error)
}

// SupplierFilter narrows All results. Zero fields match everything.
type SupplierFilter struct {
	Status SupplierStatus
	Name   string
	Limit  int
}

func (f SupplierFilter) match(v *Supplier) bool {
	if f.Status != SupplierStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Name != "" && v.Name != f.Name {
		return false
	}
	return true
}

// MemorySupplierStore keeps suppliers in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemorySupplierStore struct {
	mu    sync.RWMutex
	items map[string]*Supplier
}

// NewMemorySupplierStore returns an empty store.
func NewMemorySupplierStore() *MemorySupplierStore {
	return &MemorySupplierStore{items: make(map[string]*Supplier)}
}

var _ SupplierStore = (*MemorySupplierStore)(nil)

func (s *MemorySupplierStore) Find(ctx context.Context, id string) (*Supplier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: supplier %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemorySupplierStore) Save(ctx context.Context, v *Supplier) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil supplier", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemorySupplierStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: supplier %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// All returns matching suppliers ordered by id.
func (s *MemorySupplierStore) All(ctx context.Context, f SupplierFilter) ([]*Supplier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*Supplier, 0, len(s.items))
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

// Len reports how many suppliers are stored.
func (s *MemorySupplierStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
