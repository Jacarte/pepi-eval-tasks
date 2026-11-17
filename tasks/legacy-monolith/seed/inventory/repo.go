package inventory

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Repository stores stock items keyed by SKU.
type Repository interface {
	Get(ctx context.Context, sku string) (*StockItem, error)
	Put(ctx context.Context, item *StockItem) error
	All(ctx context.Context) ([]*StockItem, error)
}

// MemoryRepo is an in-memory Repository.
type MemoryRepo struct {
	mu    sync.Mutex
	items map[string]*StockItem
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{items: map[string]*StockItem{}} }

func (m *MemoryRepo) Get(_ context.Context, sku string) (*StockItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it, ok := m.items[sku]
	if !ok {
		return nil, fmt.Errorf("%w: sku %q", ErrNotFound, sku)
	}
	return it.clone(), nil
}

func (m *MemoryRepo) Put(ctx context.Context, item *StockItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if item.ID == "" || item.OnHand < 0 || item.Reserved < 0 {
		return fmt.Errorf("%w: stock item %+v", ErrInvalid, *item)
	}
	m.mu.Lock()
	m.items[item.ID] = item.clone()
	m.mu.Unlock()
	return nil
}

func (m *MemoryRepo) All(_ context.Context) ([]*StockItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*StockItem, 0, len(m.items))
	for _, it := range m.items {
		out = append(out, it.clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
