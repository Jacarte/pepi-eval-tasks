package orders

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// Draft is an unsubmitted cart snapshot.
type Draft struct {
	ID         string
	CustomerID string
	Items      int
	Note       string
	Total      money.Cents
	Status     DraftStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DraftStatus is the lifecycle state of a draft.
type DraftStatus int

const (
	DraftStatusUnknown DraftStatus = iota
	DraftStatusOpen
	DraftStatusAbandoned
	DraftStatusConverted
)

var draftStatusNames = []string{"unknown", "open", "abandoned", "converted"}

func (s DraftStatus) String() string {
	if s < 0 || int(s) >= len(draftStatusNames) {
		return fmt.Sprintf("DraftStatus(%d)", int(s))
	}
	return draftStatusNames[s]
}

// ParseDraftStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseDraftStatus(v string) (DraftStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range draftStatusNames {
		if i > 0 && n == v {
			return DraftStatus(i), nil
		}
	}
	return DraftStatusUnknown, fmt.Errorf("%w: unknown draft status %q", ErrInvalid, v)
}

var draftTransitions = map[DraftStatus][]DraftStatus{
	DraftStatusOpen:      {DraftStatusAbandoned, DraftStatusConverted},
	DraftStatusAbandoned: {DraftStatusConverted},
}

// CanTransition reports whether a draft may move from s to next.
func (s DraftStatus) CanTransition(next DraftStatus) bool {
	for _, t := range draftTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s DraftStatus) Terminal() bool { return len(draftTransitions[s]) == 0 }

// Validate checks the invariants every stored draft must satisfy.
func (v *Draft) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "draft", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.CustomerID) == "" {
		return &ValidationError{Entity: "draft", Field: "customer_id", Reason: "is required"}
	}
	if v.Items < 0 {
		return &ValidationError{Entity: "draft", Field: "items", Reason: "must not be negative"}
	}
	if v.Total < 0 {
		return &ValidationError{Entity: "draft", Field: "total", Reason: "must not be negative"}
	}
	if v.Status <= DraftStatusUnknown || int(v.Status) >= len(draftStatusNames) {
		return &ValidationError{Entity: "draft", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Draft) Clone() *Draft {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
