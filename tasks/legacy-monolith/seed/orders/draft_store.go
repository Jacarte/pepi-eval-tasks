package orders

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// DraftStore persists drafts.
type DraftStore interface {
	Find(ctx context.Context, id string) (*Draft, error)
	Save(ctx context.Context, v *Draft) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f DraftFilter) ([]*Draft, error)
}

// DraftFilter narrows All results. Zero fields match everything.
type DraftFilter struct {
	Status     DraftStatus
	CustomerID string
	Limit      int
}

func (f DraftFilter) match(v *Draft) bool {
	if f.Status != DraftStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.CustomerID != "" && v.CustomerID != f.CustomerID {
		return false
	}
	return true
}

// MemoryDraftStore keeps drafts in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryDraftStore struct {
	mu    sync.Mutex
	items map[string]*Draft
}

// NewMemoryDraftStore returns an empty store.
func NewMemoryDraftStore() *MemoryDraftStore {
	return &MemoryDraftStore{items: make(map[string]*Draft)}
}

var _ DraftStore = (*MemoryDraftStore)(nil)

func (st *MemoryDraftStore) Find(ctx context.Context, id string) (*Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: draft %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryDraftStore) Save(ctx context.Context, v *Draft) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil draft", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryDraftStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: draft %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// All returns matching drafts ordered by id.
func (st *MemoryDraftStore) All(ctx context.Context, f DraftFilter) ([]*Draft, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.Lock()
	out := make([]*Draft, 0, len(st.items))
	for _, v := range st.items {
		if f.match(v) {
			out = append(out, v.Clone())
		}
	}
	st.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// Len reports how many drafts are stored.
func (st *MemoryDraftStore) Len() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.items)
}
