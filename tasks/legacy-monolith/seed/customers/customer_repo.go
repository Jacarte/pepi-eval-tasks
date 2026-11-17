package customers

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// CustomerStore persists customers.
type CustomerStore interface {
	Get(ctx context.Context, id string) (*Customer, error)
	Put(ctx context.Context, v *Customer) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f CustomerFilter) ([]*Customer, error)
}

// CustomerFilter narrows List results. Zero fields match everything.
type CustomerFilter struct {
	Status CustomerStatus
	Email  string
	Limit  int
}

func (f CustomerFilter) match(v *Customer) bool {
	if f.Status != CustomerStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Email != "" && v.Email != f.Email {
		return false
	}
	return true
}

// MemoryCustomerStore keeps customers in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryCustomerStore struct {
	mu    sync.RWMutex
	items map[string]*Customer
}

// NewMemoryCustomerStore returns an empty store.
func NewMemoryCustomerStore() *MemoryCustomerStore {
	return &MemoryCustomerStore{items: make(map[string]*Customer)}
}

var _ CustomerStore = (*MemoryCustomerStore)(nil)

func (st *MemoryCustomerStore) Get(ctx context.Context, id string) (*Customer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: customer %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryCustomerStore) Put(ctx context.Context, v *Customer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil customer", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryCustomerStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: customer %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// List returns matching customers ordered by id.
func (st *MemoryCustomerStore) List(ctx context.Context, f CustomerFilter) ([]*Customer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*Customer, 0, len(st.items))
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

// Len reports how many customers are stored.
func (st *MemoryCustomerStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
