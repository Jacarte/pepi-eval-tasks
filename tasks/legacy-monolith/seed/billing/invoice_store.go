package billing

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// InvoiceStore persists invoices.
type InvoiceStore interface {
	Get(ctx context.Context, id string) (*Invoice, error)
	Put(ctx context.Context, v *Invoice) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f InvoiceFilter) ([]*Invoice, error)
}

// InvoiceFilter narrows List results. Zero fields match everything.
type InvoiceFilter struct {
	Status     InvoiceStatus
	CustomerID string
	Limit      int
}

func (f InvoiceFilter) match(v *Invoice) bool {
	if f.Status != InvoiceStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.CustomerID != "" && v.CustomerID != f.CustomerID {
		return false
	}
	return true
}

// MemoryInvoiceStore keeps invoices in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryInvoiceStore struct {
	mu    sync.RWMutex
	items map[string]*Invoice
}

// NewMemoryInvoiceStore returns an empty store.
func NewMemoryInvoiceStore() *MemoryInvoiceStore {
	return &MemoryInvoiceStore{items: make(map[string]*Invoice)}
}

var _ InvoiceStore = (*MemoryInvoiceStore)(nil)

func (s *MemoryInvoiceStore) Get(ctx context.Context, id string) (*Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: invoice %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemoryInvoiceStore) Put(ctx context.Context, v *Invoice) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil invoice", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemoryInvoiceStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: invoice %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// List returns matching invoices ordered by id.
func (s *MemoryInvoiceStore) List(ctx context.Context, f InvoiceFilter) ([]*Invoice, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*Invoice, 0, len(s.items))
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

// Len reports how many invoices are stored.
func (s *MemoryInvoiceStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
