package coupons

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// CouponStore persists coupons.
type CouponStore interface {
	Load(ctx context.Context, id string) (*Coupon, error)
	Store(ctx context.Context, v *Coupon) error
	Drop(ctx context.Context, id string) error
	Scan(ctx context.Context, f CouponFilter) ([]*Coupon, error)
}

// CouponFilter narrows Scan results. Zero fields match everything.
type CouponFilter struct {
	Status CouponStatus
	Code   string
	Limit  int
}

func (f CouponFilter) match(v *Coupon) bool {
	if f.Status != CouponStatusUnknown && v.Status != f.Status {
		return false
	}
	if f.Code != "" && v.Code != f.Code {
		return false
	}
	return true
}

// MemoryCouponStore keeps coupons in a map. It is safe for concurrent use and is
// what tests and the local dev server use.
type MemoryCouponStore struct {
	mu    sync.RWMutex
	items map[string]*Coupon
}

// NewMemoryCouponStore returns an empty store.
func NewMemoryCouponStore() *MemoryCouponStore {
	return &MemoryCouponStore{items: make(map[string]*Coupon)}
}

var _ CouponStore = (*MemoryCouponStore)(nil)

func (s *MemoryCouponStore) Load(ctx context.Context, id string) (*Coupon, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.items[id]
	if !ok {
		return nil, fmt.Errorf("%w: coupon %q", ErrNotFound, id)
	}
	return v.Clone(), nil
}

func (s *MemoryCouponStore) Store(ctx context.Context, v *Coupon) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if v == nil {
		return fmt.Errorf("%w: nil coupon", ErrInvalid)
	}
	if err := v.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[v.ID] = v.Clone()
	return nil
}

func (s *MemoryCouponStore) Drop(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("%w: coupon %q", ErrNotFound, id)
	}
	delete(s.items, id)
	return nil
}

// Scan returns matching coupons ordered by id.
func (s *MemoryCouponStore) Scan(ctx context.Context, f CouponFilter) ([]*Coupon, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	out := make([]*Coupon, 0, len(s.items))
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

// Len reports how many coupons are stored.
func (s *MemoryCouponStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}
