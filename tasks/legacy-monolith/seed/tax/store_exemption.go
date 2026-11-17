package tax

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ExemptionStore persists exemptions.
type ExemptionStore interface {
	Load(ctx context.Context, id string) (*Exemption, error)
	Store(ctx context.Context, v *Exemption) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f ExemptionFilter) ([]*Exemption, error)
}

// ExemptionFilter narrows Scan results. Zero fields match everything.
type ExemptionFilter struct {
	Status     ExemptionStatus
	CustomerID string
	Limit      int
}

func (f ExemptionFilter) match(v *Exemption) bool {
	if f.Status != ExemptionStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.CustomerID != "" && v.CustomerID != f.CustomerID {
		return false
	}
	return true
}

// MemoryExemptionStore keeps exemptions in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryExemptionStore struct {
	mu    sync.Mutex
	items map[string]*Exemption
}

// NewMemoryExemptionStore returns an empty store.
func NewMemoryExemptionStore() *MemoryExemptionStore {
	return &MemoryExemptionStore{items: make(map[string]*Exemption)}
}

var _ ExemptionStore = (*MemoryExemptionStore)(nil)

func (m *MemoryExemptionStore) Load(ctx context.Context, id string) (*Exemption, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: exemption %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryExemptionStore) Store(ctx context.Context, v *Exemption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil exemption", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryExemptionStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: exemption %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// Scan returns matching exemptions ordered by id.
func (m *MemoryExemptionStore) Scan(ctx context.Context, f ExemptionFilter) ([]*Exemption, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	out := make([]*Exemption, 0, len(m.items))
	for _, v := range m.items {
		if f.match(v) {
			out = append(out, v.Clone())
		}
	}
	m.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// Len reports how many exemptions are stored.
func (m *MemoryExemptionStore) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items)
}
