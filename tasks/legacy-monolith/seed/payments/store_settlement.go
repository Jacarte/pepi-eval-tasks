package payments

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// SettlementStore persists settlements.
type SettlementStore interface {
	Load(ctx context.Context, id string) (*Settlement, error)
	Store(ctx context.Context, v *Settlement) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f SettlementFilter) ([]*Settlement, error)
}

// SettlementFilter narrows Scan results. Zero fields match everything.
type SettlementFilter struct {
	Status SettlementStatus
	Batch  string
	Limit  int
}

func (f SettlementFilter) match(v *Settlement) bool {
	if f.Status != SettlementStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Batch != "" && v.Batch != f.Batch {
		return false
	}
	return true
}

// MemorySettlementStore keeps settlements in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemorySettlementStore struct {
	mu    sync.RWMutex
	items map[string]*Settlement
}

// NewMemorySettlementStore returns an empty store.
func NewMemorySettlementStore() *MemorySettlementStore {
	return &MemorySettlementStore{items: make(map[string]*Settlement)}
}

var _ SettlementStore = (*MemorySettlementStore)(nil)

func (m *MemorySettlementStore) Load(ctx context.Context, id string) (*Settlement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: settlement %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemorySettlementStore) Store(ctx context.Context, v *Settlement) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil settlement", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemorySettlementStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: settlement %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// Scan returns matching settlements ordered by id.
func (m *MemorySettlementStore) Scan(ctx context.Context, f SettlementFilter) ([]*Settlement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Settlement, 0, len(m.items))
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

// Len reports how many settlements are stored.
func (m *MemorySettlementStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
