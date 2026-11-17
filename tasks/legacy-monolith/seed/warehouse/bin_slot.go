package warehouse

import (
	"fmt"
	"strings"
	"time"
)

// BinSlot is a physical storage slot.
type BinSlot struct {
	ID        string
	Zone      string
	Aisle     int
	Level     int
	SKU       string // allocated SKU, if any
	Capacity  int    // units
	Status    BinSlotStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BinSlotStatus is the lifecycle state of a bin slot.
type BinSlotStatus int

const (
	BinSlotStatusUnknown BinSlotStatus = iota
	BinSlotStatusFree
	BinSlotStatusAllocated
	BinSlotStatusBlocked
)

var binSlotStatusNames = []string{"unknown", "free", "allocated", "blocked"}

func (s BinSlotStatus) String() string {
	if s < 0 || int(s) >= len(binSlotStatusNames) {
		return fmt.Sprintf("BinSlotStatus(%d)", int(s))
	}
	return binSlotStatusNames[s]
}

// ParseBinSlotStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseBinSlotStatus(v string) (BinSlotStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range binSlotStatusNames {
		if i > 0 && n == v {
			return BinSlotStatus(i), nil
		}
	}
	return BinSlotStatusUnknown, fmt.Errorf("%w: unknown bin slot status %q", ErrInvalid, v)
}

var binSlotTransitions = map[BinSlotStatus][]BinSlotStatus{
	BinSlotStatusFree:      {BinSlotStatusAllocated, BinSlotStatusBlocked},
	BinSlotStatusAllocated: {BinSlotStatusBlocked},
}

// CanTransition reports whether a bin slot may move from s to next.
func (s BinSlotStatus) CanTransition(next BinSlotStatus) bool {
	for _, t := range binSlotTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s BinSlotStatus) Terminal() bool { return len(binSlotTransitions[s]) == 0 }

// Validate checks the invariants every stored bin slot must satisfy.
func (v *BinSlot) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "bin_slot", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Zone) == "" {
		return &ValidationError{Entity: "bin_slot", Field: "zone", Reason: "is required"}
	}
	if v.Aisle < 0 {
		return &ValidationError{Entity: "bin_slot", Field: "aisle", Reason: "must not be negative"}
	}
	if v.Level < 0 {
		return &ValidationError{Entity: "bin_slot", Field: "level", Reason: "must not be negative"}
	}
	if v.Capacity < 0 {
		return &ValidationError{Entity: "bin_slot", Field: "capacity", Reason: "must not be negative"}
	}
	if v.Status <= BinSlotStatusUnknown || int(v.Status) >= len(binSlotStatusNames) {
		return &ValidationError{Entity: "bin_slot", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *BinSlot) Clone() *BinSlot {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
