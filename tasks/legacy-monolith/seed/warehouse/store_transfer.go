package warehouse

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// TransferStore persists transfers.
type TransferStore interface {
	Load(ctx context.Context, id string) (*Transfer, error)
	Store(ctx context.Context, v *Transfer) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f TransferFilter) ([]*Transfer, error)
}

// TransferFilter narrows Scan results. Zero fields match everything.
type TransferFilter struct {
	Status  TransferStatus
	FromBin string
	Limit   int
}

func (f TransferFilter) match(v *Transfer) bool {
	if f.Status != TransferStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.FromBin != "" && v.FromBin != f.FromBin {
		return false
	}
	return true
}

// MemoryTransferStore keeps transfers in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryTransferStore struct {
	mu    sync.RWMutex
	items map[string]*Transfer
}

// NewMemoryTransferStore returns an empty store.
func NewMemoryTransferStore() *MemoryTransferStore {
	return &MemoryTransferStore{items: make(map[string]*Transfer)}
}

var _ TransferStore = (*MemoryTransferStore)(nil)

func (st *MemoryTransferStore) Load(ctx context.Context, id string) (*Transfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: transfer %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryTransferStore) Store(ctx context.Context, v *Transfer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil transfer", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryTransferStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: transfer %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// Scan returns matching transfers ordered by id.
func (st *MemoryTransferStore) Scan(ctx context.Context, f TransferFilter) ([]*Transfer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*Transfer, 0, len(st.items))
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

// Len reports how many transfers are stored.
func (st *MemoryTransferStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
