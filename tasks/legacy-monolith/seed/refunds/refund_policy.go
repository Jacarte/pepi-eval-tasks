package refunds

import (
	"fmt"
	"strings"
	"time"
)

// RefundPolicy describes the refund window for a product family.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type RefundPolicy struct {
	ID              string
	Name            string
	WindowDays      int
	RestockFeePct   int
	RequiresReceipt bool
	Status          RefundPolicyStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// RefundPolicyStatus is the lifecycle state of a refund policy.
type RefundPolicyStatus int

const (
	RefundPolicyStatusUnknown RefundPolicyStatus = iota
	RefundPolicyStatusActive
	RefundPolicyStatusArchived
)

var refundPolicyStatusNames = []string{"unknown", "active", "archived"}

func (s RefundPolicyStatus) String() string {
	if s < 0 || int(s) >= len(refundPolicyStatusNames) {
		return fmt.Sprintf("RefundPolicyStatus(%d)", int(s))
	}
	return refundPolicyStatusNames[s]
}

// ParseRefundPolicyStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseRefundPolicyStatus(v string) (RefundPolicyStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range refundPolicyStatusNames {
		if i > 0 && n == v {
			return RefundPolicyStatus(i), nil
		}
	}
	return RefundPolicyStatusUnknown, fmt.Errorf("%w: unknown refund policy status %q", ErrInvalid, v)
}

var refundPolicyTransitions = map[RefundPolicyStatus][]RefundPolicyStatus{
	RefundPolicyStatusActive: {RefundPolicyStatusArchived},
}

// CanTransition reports whether a refund policy may move from s to next.
func (s RefundPolicyStatus) CanTransition(next RefundPolicyStatus) bool {
	for _, t := range refundPolicyTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s RefundPolicyStatus) Terminal() bool { return len(refundPolicyTransitions[s]) == 0 }

// Validate checks the invariants every stored refund policy must satisfy.
func (v *RefundPolicy) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "refund_policy", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Name) == "" {
		return &ValidationError{Entity: "refund_policy", Field: "name", Reason: "is required"}
	}
	if v.WindowDays < 0 {
		return &ValidationError{Entity: "refund_policy", Field: "window_days", Reason: "must not be negative"}
	}
	if v.RestockFeePct < 0 {
		return &ValidationError{Entity: "refund_policy", Field: "restock_fee_pct", Reason: "must not be negative"}
	}
	if v.Status <= RefundPolicyStatusUnknown || int(v.Status) >= len(refundPolicyStatusNames) {
		return &ValidationError{Entity: "refund_policy", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *RefundPolicy) Clone() *RefundPolicy {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
