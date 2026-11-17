package pricing

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// PriceRuleStore persists price rules.
type PriceRuleStore interface {
	Get(ctx context.Context, id string) (*PriceRule, error)
	Put(ctx context.Context, v *PriceRule) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f PriceRuleFilter) ([]*PriceRule, error)
}

// PriceRuleFilter narrows List results. Zero fields match everything.
type PriceRuleFilter struct {
	Status PriceRuleStatus
	SKU    string
	Limit  int
}

func (f PriceRuleFilter) match(v *PriceRule) bool {
	if f.Status != PriceRuleStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.SKU != "" && v.SKU != f.SKU {
		return false
	}
	return true
}

// MemoryPriceRuleStore keeps price rules in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryPriceRuleStore struct {
	mu    sync.RWMutex
	items map[string]*PriceRule
}

// NewMemoryPriceRuleStore returns an empty store.
func NewMemoryPriceRuleStore() *MemoryPriceRuleStore {
	return &MemoryPriceRuleStore{items: make(map[string]*PriceRule)}
}

var _ PriceRuleStore = (*MemoryPriceRuleStore)(nil)

func (st *MemoryPriceRuleStore) Get(ctx context.Context, id string) (*PriceRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: price rule %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryPriceRuleStore) Put(ctx context.Context, v *PriceRule) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil price rule", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryPriceRuleStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: price rule %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// List returns matching price rules ordered by id.
func (st *MemoryPriceRuleStore) List(ctx context.Context, f PriceRuleFilter) ([]*PriceRule, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*PriceRule, 0, len(st.items))
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

// Len reports how many pricerules are stored.
func (st *MemoryPriceRuleStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
