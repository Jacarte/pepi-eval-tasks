package scheduling

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// SlotStore persists slots.
type SlotStore interface {
	Load(ctx context.Context, id string) (*Slot, error)
	Store(ctx context.Context, v *Slot) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f SlotFilter) ([]*Slot, error)
}

// SlotFilter narrows Scan results. Zero fields match everything.
type SlotFilter struct {
	Status    SlotStatus
	Warehouse string
	Limit     int
}

func (f SlotFilter) match(v *Slot) bool {
	if f.Status != SlotStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Warehouse != "" && v.Warehouse != f.Warehouse {
		return false
	}
	return true
}

// MemorySlotStore keeps slots in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemorySlotStore struct {
	mu    sync.RWMutex
	items map[string]*Slot
}

// NewMemorySlotStore returns an empty store.
func NewMemorySlotStore() *MemorySlotStore {
	return &MemorySlotStore{items: make(map[string]*Slot)}
}

var _ SlotStore = (*MemorySlotStore)(nil)

func (m *MemorySlotStore) Load(ctx context.Context, id string) (*Slot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: slot %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemorySlotStore) Store(ctx context.Context, v *Slot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil slot", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemorySlotStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: slot %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// Scan returns matching slots ordered by id.
func (m *MemorySlotStore) Scan(ctx context.Context, f SlotFilter) ([]*Slot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Slot, 0, len(m.items))
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

// Len reports how many slots are stored.
func (m *MemorySlotStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
