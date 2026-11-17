package loyalty

import (
	"fmt"
	"strings"
	"time"
)

// Tier is a loyalty tier definition.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Tier struct {
	ID         string
	Name       string
	Threshold  int64 // lifetime points needed
	Multiplier int   // accrual multiplier in percent
	Status     TierStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TierStatus is the lifecycle state of a tier.
type TierStatus int

const (
	TierStatusUnknown TierStatus = iota
	TierStatusActive
	TierStatusRetired
)

var tierStatusNames = []string{"unknown", "active", "retired"}

func (s TierStatus) String() string {
	if s < 0 || int(s) >= len(tierStatusNames) {
		return fmt.Sprintf("TierStatus(%d)", int(s))
	}
	return tierStatusNames[s]
}

// ParseTierStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseTierStatus(v string) (TierStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range tierStatusNames {
		if i > 0 && n == v {
			return TierStatus(i), nil
		}
	}
	return TierStatusUnknown, fmt.Errorf("%w: unknown tier status %q", ErrInvalid, v)
}

var tierTransitions = map[TierStatus][]TierStatus{
	TierStatusActive: {TierStatusRetired},
}

// CanTransition reports whether a tier may move from s to next.
func (s TierStatus) CanTransition(next TierStatus) bool {
	for _, t := range tierTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s TierStatus) Terminal() bool { return len(tierTransitions[s]) == 0 }

// Validate checks the invariants every stored tier must satisfy.
func (v *Tier) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "tier", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Name) == "" {
		return &ValidationError{Entity: "tier", Field: "name", Reason: "is required"}
	}
	if v.Threshold < 0 {
		return &ValidationError{Entity: "tier", Field: "threshold", Reason: "must not be negative"}
	}
	if v.Multiplier < 0 {
		return &ValidationError{Entity: "tier", Field: "multiplier", Reason: "must not be negative"}
	}
	if v.Status <= TierStatusUnknown || int(v.Status) >= len(tierStatusNames) {
		return &ValidationError{Entity: "tier", Field: "status", Reason: "is not a known status"}
	}
	// TODO(lw): the export job depends on this exact behaviour (FIN-208).
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Tier) Clone() *Tier {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
