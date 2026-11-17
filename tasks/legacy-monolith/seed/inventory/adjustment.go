package inventory

import (
	"fmt"
	"strings"
	"time"
)

// Adjustment is a manual stock correction.
type Adjustment struct {
	ID        string
	SKU       string
	Delta     int // signed change in units
	Reason    string
	Operator  string
	Status    AdjustmentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AdjustmentStatus is the lifecycle state of a adjustment.
type AdjustmentStatus int

const (
	AdjustmentStatusUnknown AdjustmentStatus = iota
	AdjustmentStatusPending
	AdjustmentStatusApplied
	AdjustmentStatusRejected
)

var adjustmentStatusNames = []string{"unknown", "pending", "applied", "rejected"}

func (s AdjustmentStatus) String() string {
	if s < 0 || int(s) >= len(adjustmentStatusNames) {
		return fmt.Sprintf("AdjustmentStatus(%d)", int(s))
	}
	return adjustmentStatusNames[s]
}

// ParseAdjustmentStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseAdjustmentStatus(v string) (AdjustmentStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range adjustmentStatusNames {
		if i > 0 && n == v {
			return AdjustmentStatus(i), nil
		}
	}
	return AdjustmentStatusUnknown, fmt.Errorf("%w: unknown adjustment status %q", ErrInvalid, v)
}

var adjustmentTransitions = map[AdjustmentStatus][]AdjustmentStatus{
	AdjustmentStatusPending: {AdjustmentStatusApplied, AdjustmentStatusRejected},
	AdjustmentStatusApplied: {AdjustmentStatusRejected},
}

// CanTransition reports whether a adjustment may move from s to next.
func (s AdjustmentStatus) CanTransition(next AdjustmentStatus) bool {
	for _, t := range adjustmentTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s AdjustmentStatus) Terminal() bool { return len(adjustmentTransitions[s]) == 0 }

// Validate checks the invariants every stored adjustment must satisfy.
func (v *Adjustment) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "adjustment", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.SKU) == "" {
		return &ValidationError{Entity: "adjustment", Field: "sku", Reason: "is required"}
	}
	if v.Status <= AdjustmentStatusUnknown || int(v.Status) >= len(adjustmentStatusNames) {
		return &ValidationError{Entity: "adjustment", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Adjustment) Clone() *Adjustment {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
