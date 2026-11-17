package fraud

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ReviewStore persists reviews.
type ReviewStore interface {
	Find(ctx context.Context, id string) (*Review, error)
	Save(ctx context.Context, v *Review) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f ReviewFilter) ([]*Review, error)
}

// ReviewFilter narrows All results. Zero fields match everything.
type ReviewFilter struct {
	Status  ReviewStatus
	OrderID string
	Limit   int
}

func (f ReviewFilter) match(v *Review) bool {
	if f.Status != ReviewStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.OrderID != "" && v.OrderID != f.OrderID {
		return false
	}
	return true
}

// MemoryReviewStore keeps reviews in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryReviewStore struct {
	mu    sync.RWMutex
	items map[string]*Review
}

// NewMemoryReviewStore returns an empty store.
func NewMemoryReviewStore() *MemoryReviewStore {
	return &MemoryReviewStore{items: make(map[string]*Review)}
}

var _ ReviewStore = (*MemoryReviewStore)(nil)

func (m *MemoryReviewStore) Find(ctx context.Context, id string) (*Review, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: review %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryReviewStore) Save(ctx context.Context, v *Review) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil review", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryReviewStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: review %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// All returns matching reviews ordered by id.
func (m *MemoryReviewStore) All(ctx context.Context, f ReviewFilter) ([]*Review, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Review, 0, len(m.items))
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

// Len reports how many reviews are stored.
func (m *MemoryReviewStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
