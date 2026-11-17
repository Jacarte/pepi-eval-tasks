package customers

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// Customer is a registered shopper.
type Customer struct {
	ID            string
	Email         string
	DisplayName   string
	Segment       string // crm segment code
	LifetimeValue money.Cents
	Marketing     bool // opted into marketing email
	Status        CustomerStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CustomerStatus is the lifecycle state of a customer.
type CustomerStatus int

const (
	CustomerStatusUnknown CustomerStatus = iota
	CustomerStatusPending
	CustomerStatusActive
	CustomerStatusSuspended
	CustomerStatusClosed
)

var customerStatusNames = []string{"unknown", "pending", "active", "suspended", "closed"}

func (s CustomerStatus) String() string {
	if s < 0 || int(s) >= len(customerStatusNames) {
		return fmt.Sprintf("CustomerStatus(%d)", int(s))
	}
	return customerStatusNames[s]
}

// ParseCustomerStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseCustomerStatus(v string) (CustomerStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range customerStatusNames {
		if i > 0 && n == v {
			return CustomerStatus(i), nil
		}
	}
	return CustomerStatusUnknown, fmt.Errorf("%w: unknown customer status %q", ErrInvalid, v)
}

var customerTransitions = map[CustomerStatus][]CustomerStatus{
	CustomerStatusPending:   {CustomerStatusActive, CustomerStatusClosed},
	CustomerStatusActive:    {CustomerStatusSuspended, CustomerStatusClosed},
	CustomerStatusSuspended: {CustomerStatusClosed},
}

// CanTransition reports whether a customer may move from s to next.
func (s CustomerStatus) CanTransition(next CustomerStatus) bool {
	for _, t := range customerTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s CustomerStatus) Terminal() bool { return len(customerTransitions[s]) == 0 }

// Validate checks the invariants every stored customer must satisfy.
func (v *Customer) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "customer", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Email) == "" {
		return &ValidationError{Entity: "customer", Field: "email", Reason: "is required"}
	}
	if v.LifetimeValue < 0 {
		return &ValidationError{Entity: "customer", Field: "lifetime_value", Reason: "must not be negative"}
	}
	if v.Status <= CustomerStatusUnknown || int(v.Status) >= len(customerStatusNames) {
		return &ValidationError{Entity: "customer", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Customer) Clone() *Customer {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
