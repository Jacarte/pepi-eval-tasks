package pricing

import (
	"fmt"
	"strings"
	"time"
)

// PriceRule is a price adjustment rule.
type PriceRule struct {
	ID        string
	SKU       string // SKU or * for all
	Kind      string // percent or fixed
	Value     int64  // basis points for percent, cents for fixed
	Priority  int    // lower runs first
	Stackable bool
	Status    PriceRuleStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PriceRuleStatus is the lifecycle state of a price rule.
type PriceRuleStatus int

const (
	PriceRuleStatusUnknown PriceRuleStatus = iota
	PriceRuleStatusDraft
	PriceRuleStatusActive
	PriceRuleStatusExpired
)

var priceRuleStatusNames = []string{"unknown", "draft", "active", "expired"}

func (s PriceRuleStatus) String() string {
	if s < 0 || int(s) >= len(priceRuleStatusNames) {
		return fmt.Sprintf("PriceRuleStatus(%d)", int(s))
	}
	return priceRuleStatusNames[s]
}

// ParsePriceRuleStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParsePriceRuleStatus(v string) (PriceRuleStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range priceRuleStatusNames {
		if i > 0 && n == v {
			return PriceRuleStatus(i), nil
		}
	}
	return PriceRuleStatusUnknown, fmt.Errorf("%w: unknown price rule status %q", ErrInvalid, v)
}

var priceRuleTransitions = map[PriceRuleStatus][]PriceRuleStatus{
	PriceRuleStatusDraft:  {PriceRuleStatusActive, PriceRuleStatusExpired},
	PriceRuleStatusActive: {PriceRuleStatusExpired},
}

// CanTransition reports whether a price rule may move from s to next.
func (s PriceRuleStatus) CanTransition(next PriceRuleStatus) bool {
	for _, t := range priceRuleTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s PriceRuleStatus) Terminal() bool { return len(priceRuleTransitions[s]) == 0 }

// Validate checks the invariants every stored price rule must satisfy.
func (v *PriceRule) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "price_rule", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.SKU) == "" {
		return &ValidationError{Entity: "price_rule", Field: "sku", Reason: "is required"}
	}
	if strings.TrimSpace(v.Kind) == "" {
		return &ValidationError{Entity: "price_rule", Field: "kind", Reason: "is required"}
	}
	if v.Value < 0 {
		return &ValidationError{Entity: "price_rule", Field: "value", Reason: "must not be negative"}
	}
	if v.Priority < 0 {
		return &ValidationError{Entity: "price_rule", Field: "priority", Reason: "must not be negative"}
	}
	if v.Status <= PriceRuleStatusUnknown || int(v.Status) >= len(priceRuleStatusNames) {
		return &ValidationError{Entity: "price_rule", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *PriceRule) Clone() *PriceRule {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
