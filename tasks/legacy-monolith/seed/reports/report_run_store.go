package reports

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// ReportRunStore persists report runs.
type ReportRunStore interface {
	Find(ctx context.Context, id string) (*ReportRun, error)
	Save(ctx context.Context, v *ReportRun) error
	Remove(ctx context.Context, id string) error
	All(ctx context.Context, f ReportRunFilter) ([]*ReportRun, error)
}

// ReportRunFilter narrows All results. Zero fields match everything.
type ReportRunFilter struct {
	Status ReportRunStatus
	Kind   string
	Limit  int
}

func (f ReportRunFilter) match(v *ReportRun) bool {
	if f.Status != ReportRunStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Kind != "" && v.Kind != f.Kind {
		return false
	}
	return true
}

// MemoryReportRunStore keeps report runs in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryReportRunStore struct {
	mu    sync.RWMutex
	items map[string]*ReportRun
}

// NewMemoryReportRunStore returns an empty store.
func NewMemoryReportRunStore() *MemoryReportRunStore {
	return &MemoryReportRunStore{items: make(map[string]*ReportRun)}
}

var _ ReportRunStore = (*MemoryReportRunStore)(nil)

func (s *MemoryReportRunStore) Find(ctx context.Context, id string) (*ReportRun, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: report run %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemoryReportRunStore) Save(ctx context.Context, v *ReportRun) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil report run", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemoryReportRunStore) Remove(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: report run %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// All returns matching report runs ordered by id.
func (s *MemoryReportRunStore) All(ctx context.Context, f ReportRunFilter) ([]*ReportRun, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*ReportRun, 0, len(s.items))
	for _, v := range s.items {
		if f.match(v) {
			out = append(out, v.Clone())
		}
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

// Len reports how many reportruns are stored.
func (s *MemoryReportRunStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
