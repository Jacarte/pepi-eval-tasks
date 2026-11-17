package warehouse

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// BinSlotStore persists bin slots.
type BinSlotStore interface {
	Load(ctx context.Context, id string) (*BinSlot, error)
	Store(ctx context.Context, v *BinSlot) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f BinSlotFilter) ([]*BinSlot, error)
}

// BinSlotFilter narrows Scan results. Zero fields match everything.
type BinSlotFilter struct {
	Status BinSlotStatus
	Zone   string
	Limit  int
}

func (f BinSlotFilter) match(v *BinSlot) bool {
	if f.Status != BinSlotStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Zone != "" && v.Zone != f.Zone {
		return false
	}
	return true
}

// MemoryBinSlotStore keeps bin slots in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryBinSlotStore struct {
	mu    sync.RWMutex
	items map[string]*BinSlot
}

// NewMemoryBinSlotStore returns an empty store.
func NewMemoryBinSlotStore() *MemoryBinSlotStore {
	return &MemoryBinSlotStore{items: make(map[string]*BinSlot)}
}

var _ BinSlotStore = (*MemoryBinSlotStore)(nil)

func (st *MemoryBinSlotStore) Load(ctx context.Context, id string) (*BinSlot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: bin slot %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryBinSlotStore) Store(ctx context.Context, v *BinSlot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil bin slot", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryBinSlotStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: bin slot %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// Scan returns matching bin slots ordered by id.
func (st *MemoryBinSlotStore) Scan(ctx context.Context, f BinSlotFilter) ([]*BinSlot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*BinSlot, 0, len(st.items))
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

// Len reports how many binslots are stored.
func (st *MemoryBinSlotStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
