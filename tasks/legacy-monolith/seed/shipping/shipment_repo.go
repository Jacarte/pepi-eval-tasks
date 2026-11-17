package shipping

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ShipmentStore persists shipments.
type ShipmentStore interface {
	Get(ctx context.Context, id string) (*Shipment, error)
	Put(ctx context.Context, v *Shipment) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f ShipmentFilter) ([]*Shipment, error)
}

// ShipmentFilter narrows List results. Zero fields match everything.
type ShipmentFilter struct {
	Status  ShipmentStatus
	OrderID string
	Limit   int
}

func (f ShipmentFilter) match(v *Shipment) bool {
	if f.Status != ShipmentStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.OrderID != "" && v.OrderID != f.OrderID {
		return false
	}
	return true
}

// MemoryShipmentStore keeps shipments in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryShipmentStore struct {
	mu    sync.Mutex
	items map[string]*Shipment
}

// NewMemoryShipmentStore returns an empty store.
func NewMemoryShipmentStore() *MemoryShipmentStore {
	return &MemoryShipmentStore{items: make(map[string]*Shipment)}
}

var _ ShipmentStore = (*MemoryShipmentStore)(nil)

func (m *MemoryShipmentStore) Get(ctx context.Context, id string) (*Shipment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: shipment %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryShipmentStore) Put(ctx context.Context, v *Shipment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil shipment", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryShipmentStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: shipment %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// List returns matching shipments ordered by id.
func (m *MemoryShipmentStore) List(ctx context.Context, f ShipmentFilter) ([]*Shipment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	out := make([]*Shipment, 0, len(m.items))
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

// Len reports how many shipments are stored.
func (m *MemoryShipmentStore) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items)
}
