package searchindex

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// DocumentStore persists documents.
type DocumentStore interface {
	Get(ctx context.Context, id string) (*Document, error)
	Put(ctx context.Context, v *Document) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f DocumentFilter) ([]*Document, error)
}

// DocumentFilter narrows List results. Zero fields match everything.
type DocumentFilter struct {
	Status DocumentStatus
	Kind   string
	Limit  int
}

func (f DocumentFilter) match(v *Document) bool {
	if f.Status != DocumentStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Kind != "" && v.Kind != f.Kind {
		return false
	}
	return true
}

// MemoryDocumentStore keeps documents in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryDocumentStore struct {
	mu    sync.RWMutex
	items map[string]*Document
}

// NewMemoryDocumentStore returns an empty store.
func NewMemoryDocumentStore() *MemoryDocumentStore {
	return &MemoryDocumentStore{items: make(map[string]*Document)}
}

var _ DocumentStore = (*MemoryDocumentStore)(nil)

func (m *MemoryDocumentStore) Get(ctx context.Context, id string) (*Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: document %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryDocumentStore) Put(ctx context.Context, v *Document) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil document", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryDocumentStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: document %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// List returns matching documents ordered by id.
func (m *MemoryDocumentStore) List(ctx context.Context, f DocumentFilter) ([]*Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Document, 0, len(m.items))
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

// Len reports how many documents are stored.
func (m *MemoryDocumentStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
