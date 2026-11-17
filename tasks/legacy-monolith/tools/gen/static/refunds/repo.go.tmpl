package refunds

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Repository stores refunds. Insert must fail with ErrConflict for an id
// that already exists; Update must fail with ErrNotFound for unknown ids.
type Repository interface {
	Insert(ctx context.Context, r *Refund) error
	Update(ctx context.Context, r *Refund) error
	Get(ctx context.Context, id string) (*Refund, error)
	ForPayment(ctx context.Context, paymentID string) ([]*Refund, error)
}

type MemoryRepo struct {
	mu      sync.RWMutex
	refunds map[string]*Refund
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{refunds: map[string]*Refund{}} }

func (m *MemoryRepo) Insert(ctx context.Context, r *Refund) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.refunds[r.ID]; ok {
		return fmt.Errorf("%w: %s", ErrConflict, r.ID)
	}
	m.refunds[r.ID] = r.Clone()
	return nil
}

func (m *MemoryRepo) Update(ctx context.Context, r *Refund) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.refunds[r.ID]; !ok {
		return fmt.Errorf("%w: refund %q", ErrNotFound, r.ID)
	}
	m.refunds[r.ID] = r.Clone()
	return nil
}

func (m *MemoryRepo) Get(_ context.Context, id string) (*Refund, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.refunds[id]
	if !ok {
		return nil, fmt.Errorf("%w: refund %q", ErrNotFound, id)
	}
	return r.Clone(), nil
}

func (m *MemoryRepo) ForPayment(_ context.Context, paymentID string) ([]*Refund, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Refund
	for _, r := range m.refunds {
		if r.PaymentID == paymentID {
			out = append(out, r.Clone())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
