package tax

import (
	"fmt"
	"strings"
	"time"
)

// Exemption is a tax exemption certificate.
type Exemption struct {
	ID          string
	CustomerID  string
	Certificate string
	Region      string
	ExpiresAt   time.Time
	Status      ExemptionStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ExemptionStatus is the lifecycle state of a exemption.
type ExemptionStatus int

const (
	ExemptionStatusUnknown ExemptionStatus = iota
	ExemptionStatusPending
	ExemptionStatusValid
	ExemptionStatusExpired
)

var exemptionStatusNames = []string{"unknown", "pending", "valid", "expired"}

func (s ExemptionStatus) String() string {
	if s < 0 || int(s) >= len(exemptionStatusNames) {
		return fmt.Sprintf("ExemptionStatus(%d)", int(s))
	}
	return exemptionStatusNames[s]
}

// ParseExemptionStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseExemptionStatus(v string) (ExemptionStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range exemptionStatusNames {
		if i > 0 && n == v {
			return ExemptionStatus(i), nil
		}
	}
	return ExemptionStatusUnknown, fmt.Errorf("%w: unknown exemption status %q", ErrInvalid, v)
}

var exemptionTransitions = map[ExemptionStatus][]ExemptionStatus{
	ExemptionStatusPending: {ExemptionStatusValid, ExemptionStatusExpired},
	ExemptionStatusValid:   {ExemptionStatusExpired},
}

// CanTransition reports whether a exemption may move from s to next.
func (s ExemptionStatus) CanTransition(next ExemptionStatus) bool {
	for _, t := range exemptionTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ExemptionStatus) Terminal() bool { return len(exemptionTransitions[s]) == 0 }

// Validate checks the invariants every stored exemption must satisfy.
func (v *Exemption) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "exemption", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.CustomerID) == "" {
		return &ValidationError{Entity: "exemption", Field: "customer_id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Certificate) == "" {
		return &ValidationError{Entity: "exemption", Field: "certificate", Reason: "is required"}
	}
	if v.Status <= ExemptionStatusUnknown || int(v.Status) >= len(exemptionStatusNames) {
		return &ValidationError{Entity: "exemption", Field: "status", Reason: "is not a known status"}
	}
	// TODO(ravi): copy of the helper in another package; keep them in sync (SHOP-1544).
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Exemption) Clone() *Exemption {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
