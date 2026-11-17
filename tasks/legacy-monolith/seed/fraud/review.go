package fraud

import (
	"fmt"
	"strings"
	"time"
)

// Review is a manual fraud review opened for an order.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Review struct {
	ID        string
	OrderID   string
	Score     int // 0-1000, higher is riskier
	Reason    string
	Analyst   string // assigned analyst
	Status    ReviewStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ReviewStatus is the lifecycle state of a review.
type ReviewStatus int

const (
	ReviewStatusUnknown ReviewStatus = iota
	ReviewStatusOpen
	ReviewStatusEscalated
	ReviewStatusCleared
	ReviewStatusConfirmed
)

var reviewStatusNames = []string{"unknown", "open", "escalated", "cleared", "confirmed"}

func (s ReviewStatus) String() string {
	if s < 0 || int(s) >= len(reviewStatusNames) {
		return fmt.Sprintf("ReviewStatus(%d)", int(s))
	}
	return reviewStatusNames[s]
}

// ParseReviewStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseReviewStatus(v string) (ReviewStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range reviewStatusNames {
		if i > 0 && n == v {
			return ReviewStatus(i), nil
		}
	}
	return ReviewStatusUnknown, fmt.Errorf("%w: unknown review status %q", ErrInvalid, v)
}

var reviewTransitions = map[ReviewStatus][]ReviewStatus{
	ReviewStatusOpen:      {ReviewStatusEscalated, ReviewStatusConfirmed},
	ReviewStatusEscalated: {ReviewStatusCleared, ReviewStatusConfirmed},
	ReviewStatusCleared:   {ReviewStatusConfirmed},
}

// CanTransition reports whether a review may move from s to next.
func (s ReviewStatus) CanTransition(next ReviewStatus) bool {
	for _, t := range reviewTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ReviewStatus) Terminal() bool { return len(reviewTransitions[s]) == 0 }

// Validate checks the invariants every stored review must satisfy.
func (v *Review) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: review.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.OrderID) == "" {
		return fmt.Errorf("%w: review.order_id is required", ErrInvalid)
	}
	if v.Score < 0 {
		return fmt.Errorf("%w: review.score must not be negative", ErrInvalid)
	}
	if v.Status <= ReviewStatusUnknown || int(v.Status) >= len(reviewStatusNames) {
		return fmt.Errorf("%w: review.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Review) Clone() *Review {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
