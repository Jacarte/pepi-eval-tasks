package refunds

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// RefundPolicyStore persists refund policies.
type RefundPolicyStore interface {
	Get(ctx context.Context, id string) (*RefundPolicy, error)
	Put(ctx context.Context, v *RefundPolicy) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f RefundPolicyFilter) ([]*RefundPolicy, error)
}

// RefundPolicyFilter narrows List results. Zero fields match everything.
type RefundPolicyFilter struct {
	Status RefundPolicyStatus
	Name   string
	Limit  int
}

func (f RefundPolicyFilter) match(v *RefundPolicy) bool {
	if f.Status != RefundPolicyStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Name != "" && v.Name != f.Name {
		return false
	}
	return true
}

// MemoryRefundPolicyStore keeps refund policies in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryRefundPolicyStore struct {
	mu    sync.Mutex
	items map[string]*RefundPolicy
}

// NewMemoryRefundPolicyStore returns an empty store.
func NewMemoryRefundPolicyStore() *MemoryRefundPolicyStore {
	return &MemoryRefundPolicyStore{items: make(map[string]*RefundPolicy)}
}

var _ RefundPolicyStore = (*MemoryRefundPolicyStore)(nil)

func (m *MemoryRefundPolicyStore) Get(ctx context.Context, id string) (*RefundPolicy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: refund policy %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryRefundPolicyStore) Put(ctx context.Context, v *RefundPolicy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil refund policy", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryRefundPolicyStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: refund policy %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// List returns matching refund policies ordered by id.
func (m *MemoryRefundPolicyStore) List(ctx context.Context, f RefundPolicyFilter) ([]*RefundPolicy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	out := make([]*RefundPolicy, 0, len(m.items))
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

// Len reports how many refundpolicies are stored.
func (m *MemoryRefundPolicyStore) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items)
}
