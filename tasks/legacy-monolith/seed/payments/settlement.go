package payments

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// Settlement is a processor settlement batch.
type Settlement struct {
	ID        string
	Batch     string
	Amount    money.Cents
	Processor string
	Items     int
	Status    SettlementStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SettlementStatus is the lifecycle state of a settlement.
type SettlementStatus int

const (
	SettlementStatusUnknown SettlementStatus = iota
	SettlementStatusOpen
	SettlementStatusSubmitted
	SettlementStatusSettled
)

var settlementStatusNames = []string{"unknown", "open", "submitted", "settled"}

func (s SettlementStatus) String() string {
	if s < 0 || int(s) >= len(settlementStatusNames) {
		return fmt.Sprintf("SettlementStatus(%d)", int(s))
	}
	return settlementStatusNames[s]
}

// ParseSettlementStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseSettlementStatus(v string) (SettlementStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range settlementStatusNames {
		if i > 0 && n == v {
			return SettlementStatus(i), nil
		}
	}
	return SettlementStatusUnknown, fmt.Errorf("%w: unknown settlement status %q", ErrInvalid, v)
}

var settlementTransitions = map[SettlementStatus][]SettlementStatus{
	SettlementStatusOpen:      {SettlementStatusSubmitted, SettlementStatusSettled},
	SettlementStatusSubmitted: {SettlementStatusSettled},
}

// CanTransition reports whether a settlement may move from s to next.
func (s SettlementStatus) CanTransition(next SettlementStatus) bool {
	for _, t := range settlementTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s SettlementStatus) Terminal() bool { return len(settlementTransitions[s]) == 0 }

// Validate checks the invariants every stored settlement must satisfy.
func (v *Settlement) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "settlement", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Batch) == "" {
		return &ValidationError{Entity: "settlement", Field: "batch", Reason: "is required"}
	}
	if v.Amount < 0 {
		return &ValidationError{Entity: "settlement", Field: "amount", Reason: "must not be negative"}
	}
	if v.Items < 0 {
		return &ValidationError{Entity: "settlement", Field: "items", Reason: "must not be negative"}
	}
	if v.Status <= SettlementStatusUnknown || int(v.Status) >= len(settlementStatusNames) {
		return &ValidationError{Entity: "settlement", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Settlement) Clone() *Settlement {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
