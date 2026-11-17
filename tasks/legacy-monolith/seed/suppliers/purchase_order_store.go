package suppliers

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// PurchaseOrderStore persists purchase orders.
type PurchaseOrderStore interface {
	Find(ctx context.Context, id string) (*PurchaseOrder, error)
	Save(ctx context.Context, v *PurchaseOrder) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f PurchaseOrderFilter) ([]*PurchaseOrder, error)
}

// PurchaseOrderFilter narrows All results. Zero fields match everything.
type PurchaseOrderFilter struct {
	Status     PurchaseOrderStatus
	SupplierID string
	Limit      int
}

func (f PurchaseOrderFilter) match(v *PurchaseOrder) bool {
	if f.Status != PurchaseOrderStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.SupplierID != "" && v.SupplierID != f.SupplierID {
		return false
	}
	return true
}

// MemoryPurchaseOrderStore keeps purchase orders in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryPurchaseOrderStore struct {
	mu    sync.RWMutex
	items map[string]*PurchaseOrder
}

// NewMemoryPurchaseOrderStore returns an empty store.
func NewMemoryPurchaseOrderStore() *MemoryPurchaseOrderStore {
	return &MemoryPurchaseOrderStore{items: make(map[string]*PurchaseOrder)}
}

var _ PurchaseOrderStore = (*MemoryPurchaseOrderStore)(nil)

func (s *MemoryPurchaseOrderStore) Find(ctx context.Context, id string) (*PurchaseOrder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: purchase order %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemoryPurchaseOrderStore) Save(ctx context.Context, v *PurchaseOrder) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil purchase order", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemoryPurchaseOrderStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: purchase order %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// All returns matching purchase orders ordered by id.
func (s *MemoryPurchaseOrderStore) All(ctx context.Context, f PurchaseOrderFilter) ([]*PurchaseOrder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*PurchaseOrder, 0, len(s.items))
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

// Len reports how many purchaseorders are stored.
func (s *MemoryPurchaseOrderStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
