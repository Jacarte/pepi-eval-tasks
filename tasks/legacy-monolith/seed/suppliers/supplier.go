package suppliers

import (
	"fmt"
	"strings"
	"time"
)

// Supplier is a vendor we buy stock from.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Supplier struct {
	ID           string
	Name         string
	Country      string
	ContactEmail string
	LeadTimeDays int
	Preferred    bool
	Status       SupplierStatus
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SupplierStatus is the lifecycle state of a supplier.
type SupplierStatus int

const (
	SupplierStatusUnknown SupplierStatus = iota
	SupplierStatusProspect
	SupplierStatusOnboarded
	SupplierStatusSuspended
)

var supplierStatusNames = []string{"unknown", "prospect", "onboarded", "suspended"}

func (s SupplierStatus) String() string {
	if s < 0 || int(s) >= len(supplierStatusNames) {
		return fmt.Sprintf("SupplierStatus(%d)", int(s))
	}
	return supplierStatusNames[s]
}

// ParseSupplierStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseSupplierStatus(v string) (SupplierStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range supplierStatusNames {
		if i > 0 && n == v {
			return SupplierStatus(i), nil
		}
	}
	return SupplierStatusUnknown, fmt.Errorf("%w: unknown supplier status %q", ErrInvalid, v)
}

var supplierTransitions = map[SupplierStatus][]SupplierStatus{
	SupplierStatusProspect:  {SupplierStatusOnboarded, SupplierStatusSuspended},
	SupplierStatusOnboarded: {SupplierStatusSuspended},
}

// CanTransition reports whether a supplier may move from s to next.
func (s SupplierStatus) CanTransition(next SupplierStatus) bool {
	for _, t := range supplierTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s SupplierStatus) Terminal() bool { return len(supplierTransitions[s]) == 0 }

// Validate checks the invariants every stored supplier must satisfy.
func (v *Supplier) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: supplier.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Name) == "" {
		return fmt.Errorf("%w: supplier.name is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Country) == "" {
		return fmt.Errorf("%w: supplier.country is required", ErrInvalid)
	}
	if v.LeadTimeDays < 0 {
		return fmt.Errorf("%w: supplier.lead_time_days must not be negative", ErrInvalid)
	}
	if v.Status <= SupplierStatusUnknown || int(v.Status) >= len(supplierStatusNames) {
		return fmt.Errorf("%w: supplier.status is not a known status", ErrInvalid)
	}
	// NOTE(pc): remove the special case below when ops confirms.
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Supplier) Clone() *Supplier {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
