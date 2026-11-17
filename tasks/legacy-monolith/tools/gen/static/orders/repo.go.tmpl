package orders

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Repository stores orders.
type Repository interface {
	Get(ctx context.Context, id string) (*Order, error)
	Save(ctx context.Context, o *Order) error
	ListByCustomer(ctx context.Context, customerID string) ([]*Order, error)
}

// MemoryRepo is the in-memory Repository used by tests and the dev server.
type MemoryRepo struct {
	mu     sync.RWMutex
	orders map[string]*Order
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{orders: make(map[string]*Order)} }

func (r *MemoryRepo) Get(ctx context.Context, id string) (*Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, ok := r.orders[id]
	if !ok {
		return nil, fmt.Errorf("%w: order %q", ErrNotFound, id)
	}
	return o.Clone(), nil
}

func (r *MemoryRepo) Save(ctx context.Context, o *Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if o == nil || o.ID == "" {
		return fmt.Errorf("%w: order without id", ErrInvalid)
	}
	r.mu.Lock()
	r.orders[o.ID] = o.Clone()
	r.mu.Unlock()
	return nil
}

func (r *MemoryRepo) ListByCustomer(ctx context.Context, customerID string) ([]*Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*Order
	for _, o := range r.orders {
		if o.CustomerID == customerID {
			out = append(out, o.Clone())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
