package fraud

import (
	"fmt"
	"strings"
	"time"
)

// Rule is a weighted scoring rule.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Rule struct {
	ID         string
	Name       string
	Weight     int
	Enabled    bool
	Expression string // legacy rule DSL, see scorer.go
	Status     RuleStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// RuleStatus is the lifecycle state of a rule.
type RuleStatus int

const (
	RuleStatusUnknown RuleStatus = iota
	RuleStatusDraft
	RuleStatusLive
	RuleStatusRetired
)

var ruleStatusNames = []string{"unknown", "draft", "live", "retired"}

func (s RuleStatus) String() string {
	if s < 0 || int(s) >= len(ruleStatusNames) {
		return fmt.Sprintf("RuleStatus(%d)", int(s))
	}
	return ruleStatusNames[s]
}

// ParseRuleStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseRuleStatus(v string) (RuleStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range ruleStatusNames {
		if i > 0 && n == v {
			return RuleStatus(i), nil
		}
	}
	return RuleStatusUnknown, fmt.Errorf("%w: unknown rule status %q", ErrInvalid, v)
}

var ruleTransitions = map[RuleStatus][]RuleStatus{
	RuleStatusDraft: {RuleStatusLive, RuleStatusRetired},
	RuleStatusLive:  {RuleStatusRetired},
}

// CanTransition reports whether a rule may move from s to next.
func (s RuleStatus) CanTransition(next RuleStatus) bool {
	for _, t := range ruleTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s RuleStatus) Terminal() bool { return len(ruleTransitions[s]) == 0 }

// Validate checks the invariants every stored rule must satisfy.
func (v *Rule) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: rule.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Name) == "" {
		return fmt.Errorf("%w: rule.name is required", ErrInvalid)
	}
	if v.Weight < 0 {
		return fmt.Errorf("%w: rule.weight must not be negative", ErrInvalid)
	}
	if v.Status <= RuleStatusUnknown || int(v.Status) >= len(ruleStatusNames) {
		return fmt.Errorf("%w: rule.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Rule) Clone() *Rule {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
