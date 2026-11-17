package loyalty

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// TierStore persists tiers.
type TierStore interface {
	Find(ctx context.Context, id string) (*Tier, error)
	Save(ctx context.Context, v *Tier) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f TierFilter) ([]*Tier, error)
}

// TierFilter narrows All results. Zero fields match everything.
type TierFilter struct {
	Status TierStatus
	Name   string
	Limit  int
}

func (f TierFilter) match(v *Tier) bool {
	if f.Status != TierStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Name != "" && v.Name != f.Name {
		return false
	}
	return true
}

// MemoryTierStore keeps tiers in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryTierStore struct {
	mu    sync.RWMutex
	items map[string]*Tier
}

// NewMemoryTierStore returns an empty store.
func NewMemoryTierStore() *MemoryTierStore {
	return &MemoryTierStore{items: make(map[string]*Tier)}
}

var _ TierStore = (*MemoryTierStore)(nil)

func (m *MemoryTierStore) Find(ctx context.Context, id string) (*Tier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: tier %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryTierStore) Save(ctx context.Context, v *Tier) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil tier", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryTierStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: tier %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// All returns matching tiers ordered by id.
func (m *MemoryTierStore) All(ctx context.Context, f TierFilter) ([]*Tier, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Tier, 0, len(m.items))
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

// Len reports how many tiers are stored.
func (m *MemoryTierStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
