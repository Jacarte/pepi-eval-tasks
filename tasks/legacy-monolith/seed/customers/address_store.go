package customers

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// AddressStore persists addresses.
type AddressStore interface {
	Get(ctx context.Context, id string) (*Address, error)
	Put(ctx context.Context, v *Address) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f AddressFilter) ([]*Address, error)
}

// AddressFilter narrows List results. Zero fields match everything.
type AddressFilter struct {
	Status     AddressStatus
	CustomerID string
	Limit      int
}

func (f AddressFilter) match(v *Address) bool {
	if f.Status != AddressStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.CustomerID != "" && v.CustomerID != f.CustomerID {
		return false
	}
	return true
}

// MemoryAddressStore keeps addresses in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryAddressStore struct {
	mu    sync.RWMutex
	items map[string]*Address
}

// NewMemoryAddressStore returns an empty store.
func NewMemoryAddressStore() *MemoryAddressStore {
	return &MemoryAddressStore{items: make(map[string]*Address)}
}

var _ AddressStore = (*MemoryAddressStore)(nil)

func (st *MemoryAddressStore) Get(ctx context.Context, id string) (*Address, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: address %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryAddressStore) Put(ctx context.Context, v *Address) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil address", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryAddressStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: address %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// List returns matching addresses ordered by id.
func (st *MemoryAddressStore) List(ctx context.Context, f AddressFilter) ([]*Address, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*Address, 0, len(st.items))
	for _, v := range st.items {
		if f.match(v) {
			out = append(out, v.Clone())
		}
	}
	st.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// Len reports how many addresses are stored.
func (st *MemoryAddressStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
