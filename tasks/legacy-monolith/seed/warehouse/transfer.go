package warehouse

import (
	"fmt"
	"strings"
	"time"
)

// Transfer moves stock between two bins.
type Transfer struct {
	ID        string
	FromBin   string
	ToBin     string
	SKU       string
	Qty       int
	Status    TransferStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TransferStatus is the lifecycle state of a transfer.
type TransferStatus int

const (
	TransferStatusUnknown TransferStatus = iota
	TransferStatusPlanned
	TransferStatusMoving
	TransferStatusDone
)

var transferStatusNames = []string{"unknown", "planned", "moving", "done"}

func (s TransferStatus) String() string {
	if s < 0 || int(s) >= len(transferStatusNames) {
		return fmt.Sprintf("TransferStatus(%d)", int(s))
	}
	return transferStatusNames[s]
}

// ParseTransferStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseTransferStatus(v string) (TransferStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range transferStatusNames {
		if i > 0 && n == v {
			return TransferStatus(i), nil
		}
	}
	return TransferStatusUnknown, fmt.Errorf("%w: unknown transfer status %q", ErrInvalid, v)
}

var transferTransitions = map[TransferStatus][]TransferStatus{
	TransferStatusPlanned: {TransferStatusMoving, TransferStatusDone},
	TransferStatusMoving:  {TransferStatusDone},
}

// CanTransition reports whether a transfer may move from s to next.
func (s TransferStatus) CanTransition(next TransferStatus) bool {
	for _, t := range transferTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s TransferStatus) Terminal() bool { return len(transferTransitions[s]) == 0 }

// Validate checks the invariants every stored transfer must satisfy.
func (v *Transfer) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "transfer", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.FromBin) == "" {
		return &ValidationError{Entity: "transfer", Field: "from_bin", Reason: "is required"}
	}
	if strings.TrimSpace(v.ToBin) == "" {
		return &ValidationError{Entity: "transfer", Field: "to_bin", Reason: "is required"}
	}
	if strings.TrimSpace(v.SKU) == "" {
		return &ValidationError{Entity: "transfer", Field: "sku", Reason: "is required"}
	}
	if v.Qty < 0 {
		return &ValidationError{Entity: "transfer", Field: "qty", Reason: "must not be negative"}
	}
	if v.Status <= TransferStatusUnknown || int(v.Status) >= len(transferStatusNames) {
		return &ValidationError{Entity: "transfer", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Transfer) Clone() *Transfer {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
