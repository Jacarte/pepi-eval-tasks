// Package store keeps customers and payments in memory.
package store

import (
	"fmt"
	"sync"
)

type Customer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Payment struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id"`
	AmountCents   int64  `json:"amount_cents"`
	Currency      string `json:"currency"`
	Status        string `json:"status"` // authorized, captured, voided
	RefundedCents int64  `json:"refunded_cents"`
}

// Memory is safe for concurrent use; each method is atomic on its own.
type Memory struct {
	mu        sync.Mutex
	seq       int
	customers map[string]Customer
	payments  map[string]Payment
}

func NewMemory() *Memory {
	return &Memory{customers: map[string]Customer{}, payments: map[string]Payment{}}
}

func (m *Memory) nextID(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s_%d", prefix, m.seq)
}

// AddCustomer returns false when the email is already registered.
func (m *Memory) AddCustomer(c Customer) (Customer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.customers {
		if existing.Email == c.Email {
			return Customer{}, false
		}
	}
	c.ID = m.nextID("cus")
	m.customers[c.ID] = c
	return c, true
}

func (m *Memory) Customer(id string) (Customer, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.customers[id]
	return c, ok
}

func (m *Memory) AddPayment(p Payment) Payment {
	m.mu.Lock()
	defer m.mu.Unlock()
	p.ID = m.nextID("pay")
	p.Status = "authorized"
	m.payments[p.ID] = p
	return p
}

func (m *Memory) Payment(id string) (Payment, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[id]
	return p, ok
}

// UpdatePayment runs fn on the stored payment under the store lock and keeps
// the result when fn returns nil. It returns false when the payment is unknown.
func (m *Memory) UpdatePayment(id string, fn func(*Payment) error) (Payment, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[id]
	if !ok {
		return Payment{}, false, nil
	}
	if err := fn(&p); err != nil {
		return p, true, err
	}
	m.payments[id] = p
	return p, true, nil
}
