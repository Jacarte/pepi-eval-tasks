package notifications

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// MessageStore persists messages.
type MessageStore interface {
	Load(ctx context.Context, id string) (*Message, error)
	Store(ctx context.Context, v *Message) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f MessageFilter) ([]*Message, error)
}

// MessageFilter narrows Scan results. Zero fields match everything.
type MessageFilter struct {
	Status    MessageStatus
	Recipient string
	Limit     int
}

func (f MessageFilter) match(v *Message) bool {
	if f.Status != MessageStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Recipient != "" && v.Recipient != f.Recipient {
		return false
	}
	return true
}

// MemoryMessageStore keeps messages in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryMessageStore struct {
	mu    sync.RWMutex
	items map[string]*Message
}

// NewMemoryMessageStore returns an empty store.
func NewMemoryMessageStore() *MemoryMessageStore {
	return &MemoryMessageStore{items: make(map[string]*Message)}
}

var _ MessageStore = (*MemoryMessageStore)(nil)

func (m *MemoryMessageStore) Load(ctx context.Context, id string) (*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: message %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (m *MemoryMessageStore) Store(ctx context.Context, v *Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil message", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[v.ID] = v.Clone()
	return nil
}

func (m *MemoryMessageStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.items[id]; !ok {
		return fmt.Errorf("%w: message %q", ErrNotFound, id)
	}
	delete(m.items, id)
	return nil
}

// Scan returns matching messages ordered by id.
func (m *MemoryMessageStore) Scan(ctx context.Context, f MessageFilter) ([]*Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.RLock()
	out := make([]*Message, 0, len(m.items))
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

// Len reports how many messages are stored.
func (m *MemoryMessageStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.items)
}
