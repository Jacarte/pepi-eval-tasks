package scheduling

import (
	"fmt"
	"strings"
	"time"
)

// Slot is a delivery booking window.
type Slot struct {
	ID        string
	Warehouse string
	Window    string // e.g. 09:00-12:00
	Capacity  int
	Booked    int
	Status    SlotStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SlotStatus is the lifecycle state of a slot.
type SlotStatus int

const (
	SlotStatusUnknown SlotStatus = iota
	SlotStatusOpen
	SlotStatusFull
	SlotStatusClosed
)

var slotStatusNames = []string{"unknown", "open", "full", "closed"}

func (s SlotStatus) String() string {
	if s < 0 || int(s) >= len(slotStatusNames) {
		return fmt.Sprintf("SlotStatus(%d)", int(s))
	}
	return slotStatusNames[s]
}

// ParseSlotStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseSlotStatus(v string) (SlotStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range slotStatusNames {
		if i > 0 && n == v {
			return SlotStatus(i), nil
		}
	}
	return SlotStatusUnknown, fmt.Errorf("%w: unknown slot status %q", ErrInvalid, v)
}

var slotTransitions = map[SlotStatus][]SlotStatus{
	SlotStatusOpen: {SlotStatusFull, SlotStatusClosed},
	SlotStatusFull: {SlotStatusClosed},
}

// CanTransition reports whether a slot may move from s to next.
func (s SlotStatus) CanTransition(next SlotStatus) bool {
	for _, t := range slotTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s SlotStatus) Terminal() bool { return len(slotTransitions[s]) == 0 }

// Validate checks the invariants every stored slot must satisfy.
func (v *Slot) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "slot", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Warehouse) == "" {
		return &ValidationError{Entity: "slot", Field: "warehouse", Reason: "is required"}
	}
	if strings.TrimSpace(v.Window) == "" {
		return &ValidationError{Entity: "slot", Field: "window", Reason: "is required"}
	}
	if v.Capacity < 0 {
		return &ValidationError{Entity: "slot", Field: "capacity", Reason: "must not be negative"}
	}
	if v.Booked < 0 {
		return &ValidationError{Entity: "slot", Field: "booked", Reason: "must not be negative"}
	}
	if v.Status <= SlotStatusUnknown || int(v.Status) >= len(slotStatusNames) {
		return &ValidationError{Entity: "slot", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Slot) Clone() *Slot {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
