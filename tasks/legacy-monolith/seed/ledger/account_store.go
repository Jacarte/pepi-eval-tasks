package ledger

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// AccountStore persists accounts.
type AccountStore interface {
	Get(ctx context.Context, id string) (*Account, error)
	Put(ctx context.Context, v *Account) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f AccountFilter) ([]*Account, error)
}

// AccountFilter narrows List results. Zero fields match everything.
type AccountFilter struct {
	Status AccountStatus
	Code   string
	Limit  int
}

func (f AccountFilter) match(v *Account) bool {
	if f.Status != AccountStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Code != "" && v.Code != f.Code {
		return false
	}
	return true
}

// MemoryAccountStore keeps accounts in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryAccountStore struct {
	mu    sync.RWMutex
	items map[string]*Account
}

// NewMemoryAccountStore returns an empty store.
func NewMemoryAccountStore() *MemoryAccountStore {
	return &MemoryAccountStore{items: make(map[string]*Account)}
}

var _ AccountStore = (*MemoryAccountStore)(nil)

func (st *MemoryAccountStore) Get(ctx context.Context, id string) (*Account, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: account %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryAccountStore) Put(ctx context.Context, v *Account) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil account", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryAccountStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: account %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// List returns matching accounts ordered by id.
func (st *MemoryAccountStore) List(ctx context.Context, f AccountFilter) ([]*Account, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*Account, 0, len(st.items))
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

// Len reports how many accounts are stored.
func (st *MemoryAccountStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
