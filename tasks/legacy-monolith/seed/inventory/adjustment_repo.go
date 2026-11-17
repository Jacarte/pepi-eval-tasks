package inventory

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// AdjustmentStore persists adjustments.
type AdjustmentStore interface {
	Get(ctx context.Context, id string) (*Adjustment, error)
	Put(ctx context.Context, v *Adjustment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f AdjustmentFilter) ([]*Adjustment, error)
}

// AdjustmentFilter narrows List results. Zero fields match everything.
type AdjustmentFilter struct {
	Status AdjustmentStatus
	SKU    string
	Limit  int
}

func (f AdjustmentFilter) match(v *Adjustment) bool {
	if f.Status != AdjustmentStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.SKU != "" && v.SKU != f.SKU {
		return false
	}
	return true
}

// MemoryAdjustmentStore keeps adjustments in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryAdjustmentStore struct {
	mu    sync.RWMutex
	items map[string]*Adjustment
}

// NewMemoryAdjustmentStore returns an empty store.
func NewMemoryAdjustmentStore() *MemoryAdjustmentStore {
	return &MemoryAdjustmentStore{items: make(map[string]*Adjustment)}
}

var _ AdjustmentStore = (*MemoryAdjustmentStore)(nil)

func (s *MemoryAdjustmentStore) Get(ctx context.Context, id string) (*Adjustment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: adjustment %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemoryAdjustmentStore) Put(ctx context.Context, v *Adjustment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil adjustment", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemoryAdjustmentStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: adjustment %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// List returns matching adjustments ordered by id.
func (s *MemoryAdjustmentStore) List(ctx context.Context, f AdjustmentFilter) ([]*Adjustment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*Adjustment, 0, len(s.items))
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

// Len reports how many adjustments are stored.
func (s *MemoryAdjustmentStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
