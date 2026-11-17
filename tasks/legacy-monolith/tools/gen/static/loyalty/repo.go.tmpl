package loyalty

import (
	"context"
	"fmt"
	"sync"
)

// Repository stores loyalty accounts.
type Repository interface {
	Load(ctx context.Context, id string) (*Account, error)
	Store(ctx context.Context, a *Account) error
}

type MemoryRepo struct {
	mu       sync.Mutex
	accounts map[string]*Account
}

func NewMemoryRepo() *MemoryRepo { return &MemoryRepo{accounts: make(map[string]*Account)} }

func (m *MemoryRepo) Load(_ context.Context, id string) (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.accounts[id]
	if !ok {
		return nil, fmt.Errorf("%w: account %q", ErrNotFound, id)
	}
	return a.clone(), nil
}

func (m *MemoryRepo) Store(ctx context.Context, a *Account) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a.Balance < 0 {
		return fmt.Errorf("%w: negative balance on %s", ErrInvalid, a.ID)
	}
	m.mu.Lock()
	m.accounts[a.ID] = a.clone()
	m.mu.Unlock()
	return nil
}
