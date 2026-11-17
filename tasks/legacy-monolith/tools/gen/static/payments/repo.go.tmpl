package payments

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Repository stores payments.
type Repository interface {
	Get(ctx context.Context, id string) (*Payment, error)
	Save(ctx context.Context, p *Payment) error
	ByOrder(ctx context.Context, orderID string) ([]*Payment, error)
}

// MemoryRepo is an in-memory Repository.
type MemoryRepo struct {
	mu   sync.Mutex
	byID map[string]*Payment
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{byID: map[string]*Payment{}} }

func (r *MemoryRepo) Get(_ context.Context, id string) (*Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("%w: payment %q", ErrNotFound, id)
	}
	return p.Clone(), nil
}

func (r *MemoryRepo) Save(ctx context.Context, p *Payment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.ID == "" {
		return fmt.Errorf("%w: payment without id", ErrInvalid)
	}
	r.mu.Lock()
	r.byID[p.ID] = p.Clone()
	r.mu.Unlock()
	return nil
}

func (r *MemoryRepo) ByOrder(_ context.Context, orderID string) ([]*Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*Payment
	for _, p := range r.byID {
		if p.OrderID == orderID {
			out = append(out, p.Clone())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
