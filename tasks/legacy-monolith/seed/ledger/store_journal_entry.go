package ledger

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// JournalEntryStore persists journal entries.
type JournalEntryStore interface {
	Get(ctx context.Context, id string) (*JournalEntry, error)
	Put(ctx context.Context, v *JournalEntry) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, f JournalEntryFilter) ([]*JournalEntry, error)
}

// JournalEntryFilter narrows List results. Zero fields match everything.
type JournalEntryFilter struct {
	Status  JournalEntryStatus
	Account string
	Limit   int
}

func (f JournalEntryFilter) match(v *JournalEntry) bool {
	if f.Status != JournalEntryStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Account != "" && v.Account != f.Account {
		return false
	}
	return true
}

// MemoryJournalEntryStore keeps journal entries in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryJournalEntryStore struct {
	mu    sync.RWMutex
	items map[string]*JournalEntry
}

// NewMemoryJournalEntryStore returns an empty store.
func NewMemoryJournalEntryStore() *MemoryJournalEntryStore {
	return &MemoryJournalEntryStore{items: make(map[string]*JournalEntry)}
}

var _ JournalEntryStore = (*MemoryJournalEntryStore)(nil)

func (st *MemoryJournalEntryStore) Get(ctx context.Context, id string) (*JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	defer st.mu.RUnlock()
	v, ok := st.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: journal entry %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (st *MemoryJournalEntryStore) Put(ctx context.Context, v *JournalEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil journal entry", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.items[v.ID] = v.Clone()
	return nil
}

func (st *MemoryJournalEntryStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if _, ok := st.items[id]; !ok {
		return fmt.Errorf("%w: journal entry %q", ErrNotFound, id)
	}
	delete(st.items, id)
	return nil
}

// List returns matching journal entries ordered by id.
func (st *MemoryJournalEntryStore) List(ctx context.Context, f JournalEntryFilter) ([]*JournalEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	st.mu.RLock()
	out := make([]*JournalEntry, 0, len(st.items))
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

// Len reports how many journalentries are stored.
func (st *MemoryJournalEntryStore) Len() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return len(st.items)
}
