package returns

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ReturnRequestStore persists return requests.
type ReturnRequestStore interface {
	Find(ctx context.Context, id string) (*ReturnRequest, error)
	Save(ctx context.Context, v *ReturnRequest) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f ReturnRequestFilter) ([]*ReturnRequest, error)
}

// ReturnRequestFilter narrows All results. Zero fields match everything.
type ReturnRequestFilter struct {
	Status  ReturnRequestStatus
	OrderID string
	Limit   int
}

func (f ReturnRequestFilter) match(v *ReturnRequest) bool {
	if f.Status != ReturnRequestStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.OrderID != "" && v.OrderID != f.OrderID {
		return false
	}
	return true
}

// MemoryReturnRequestStore keeps return requests in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryReturnRequestStore struct {
	mu    sync.Mutex
	items map[string]*ReturnRequest
}

// NewMemoryReturnRequestStore returns an empty store.
func NewMemoryReturnRequestStore() *MemoryReturnRequestStore {
	return &MemoryReturnRequestStore{items: make(map[string]*ReturnRequest)}
}

var _ ReturnRequestStore = (*MemoryReturnRequestStore)(nil)

func (st *MemoryReturnRequestStore) Find(ctx context.Context, id string) (*ReturnRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: return request %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryReturnRequestStore) Save(ctx context.Context, v *ReturnRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil return request", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryReturnRequestStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: return request %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// All returns matching return requests ordered by id.
func (st *MemoryReturnRequestStore) All(ctx context.Context, f ReturnRequestFilter) ([]*ReturnRequest, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.Lock()
	out := make([]*ReturnRequest, 0, len(st.items))
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

// Len reports how many returnrequests are stored.
func (st *MemoryReturnRequestStore) Len() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.items)
}
