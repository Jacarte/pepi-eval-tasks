package fraud

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// RuleStore persists rules.
type RuleStore interface {
	Find(ctx context.Context, id string) (*Rule, error)
	Save(ctx context.Context, v *Rule) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f RuleFilter) ([]*Rule, error)
}

// RuleFilter narrows All results. Zero fields match everything.
type RuleFilter struct {
	Status RuleStatus
	Name   string
	Limit  int
}

func (f RuleFilter) match(v *Rule) bool {
	if f.Status != RuleStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Name != "" && v.Name != f.Name {
		return false
	}
	return true
}

// MemoryRuleStore keeps rules in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryRuleStore struct {
	mu    sync.RWMutex
	items map[string]*Rule
}

// NewMemoryRuleStore returns an empty store.
func NewMemoryRuleStore() *MemoryRuleStore {
	return &MemoryRuleStore{items: make(map[string]*Rule)}
}

var _ RuleStore = (*MemoryRuleStore)(nil)

func (m *MemoryRuleStore) Find(ctx context.Context, id string) (*Rule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: rule %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryRuleStore) Save(ctx context.Context, v *Rule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil rule", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryRuleStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: rule %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// All returns matching rules ordered by id.
func (m *MemoryRuleStore) All(ctx context.Context, f RuleFilter) ([]*Rule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Rule, 0, len(m.items))
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

// Len reports how many rules are stored.
func (m *MemoryRuleStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
