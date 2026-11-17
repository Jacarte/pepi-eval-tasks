package scheduling

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// JobStore persists jobs.
type JobStore interface {
	Load(ctx context.Context, id string) (*Job, error)
	Store(ctx context.Context, v *Job) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f JobFilter) ([]*Job, error)
}

// JobFilter narrows Scan results. Zero fields match everything.
type JobFilter struct {
	Status JobStatus
	Name   string
	Limit  int
}

func (f JobFilter) match(v *Job) bool {
	if f.Status != JobStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Name != "" && v.Name != f.Name {
		return false
	}
	return true
}

// MemoryJobStore keeps jobs in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryJobStore struct {
	mu    sync.RWMutex
	items map[string]*Job
}

// NewMemoryJobStore returns an empty store.
func NewMemoryJobStore() *MemoryJobStore {
	return &MemoryJobStore{items: make(map[string]*Job)}
}

var _ JobStore = (*MemoryJobStore)(nil)

func (m *MemoryJobStore) Load(ctx context.Context, id string) (*Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: job %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryJobStore) Store(ctx context.Context, v *Job) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil job", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryJobStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: job %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// Scan returns matching jobs ordered by id.
func (m *MemoryJobStore) Scan(ctx context.Context, f JobFilter) ([]*Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Job, 0, len(m.items))
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

// Len reports how many jobs are stored.
func (m *MemoryJobStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
