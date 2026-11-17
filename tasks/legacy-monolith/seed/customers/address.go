package customers

import (
	"fmt"
	"strings"
	"time"
)

// Address is a postal address on file for a customer.
type Address struct {
	ID         string
	CustomerID string
	Line1      string
	City       string
	Postcode   string
	Country    string // ISO 3166 alpha-2
	Primary    bool
	Status     AddressStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// AddressStatus is the lifecycle state of a address.
type AddressStatus int

const (
	AddressStatusUnknown AddressStatus = iota
	AddressStatusActive
	AddressStatusArchived
)

var addressStatusNames = []string{"unknown", "active", "archived"}

func (s AddressStatus) String() string {
	if s < 0 || int(s) >= len(addressStatusNames) {
		return fmt.Sprintf("AddressStatus(%d)", int(s))
	}
	return addressStatusNames[s]
}

// ParseAddressStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseAddressStatus(v string) (AddressStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range addressStatusNames {
		if i > 0 && n == v {
			return AddressStatus(i), nil
		}
	}
	return AddressStatusUnknown, fmt.Errorf("%w: unknown address status %q", ErrInvalid, v)
}

var addressTransitions = map[AddressStatus][]AddressStatus{
	AddressStatusActive: {AddressStatusArchived},
}

// CanTransition reports whether a address may move from s to next.
func (s AddressStatus) CanTransition(next AddressStatus) bool {
	for _, t := range addressTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s AddressStatus) Terminal() bool { return len(addressTransitions[s]) == 0 }

// Validate checks the invariants every stored address must satisfy.
func (v *Address) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "address", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.CustomerID) == "" {
		return &ValidationError{Entity: "address", Field: "customer_id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Line1) == "" {
		return &ValidationError{Entity: "address", Field: "line1", Reason: "is required"}
	}
	if strings.TrimSpace(v.Country) == "" {
		return &ValidationError{Entity: "address", Field: "country", Reason: "is required"}
	}
	if v.Status <= AddressStatusUnknown || int(v.Status) >= len(addressStatusNames) {
		return &ValidationError{Entity: "address", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Address) Clone() *Address {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
