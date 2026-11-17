package catalog

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ProductListingStore persists product listings.
type ProductListingStore interface {
	Get(ctx context.Context, id string) (*ProductListing, error)
	Put(ctx context.Context, v *ProductListing) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f ProductListingFilter) ([]*ProductListing, error)
}

// ProductListingFilter narrows List results. Zero fields match everything.
type ProductListingFilter struct {
	Status ProductListingStatus
	SKU    string
	Limit  int
}

func (f ProductListingFilter) match(v *ProductListing) bool {
	if f.Status != ProductListingStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.SKU != "" && v.SKU != f.SKU {
		return false
	}
	return true
}

// MemoryProductListingStore keeps product listings in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryProductListingStore struct {
	mu    sync.RWMutex
	items map[string]*ProductListing
}

// NewMemoryProductListingStore returns an empty store.
func NewMemoryProductListingStore() *MemoryProductListingStore {
	return &MemoryProductListingStore{items: make(map[string]*ProductListing)}
}

var _ ProductListingStore = (*MemoryProductListingStore)(nil)

func (m *MemoryProductListingStore) Get(ctx context.Context, id string) (*ProductListing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: product listing %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryProductListingStore) Put(ctx context.Context, v *ProductListing) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil product listing", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryProductListingStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: product listing %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// List returns matching product listings ordered by id.
func (m *MemoryProductListingStore) List(ctx context.Context, f ProductListingFilter) ([]*ProductListing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*ProductListing, 0, len(m.items))
	for _, v := range m.items {
		if f.match(v) {
			out = append(out, v.Clone())
		}
	}
	m.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// Len reports how many productlistings are stored.
func (m *MemoryProductListingStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
