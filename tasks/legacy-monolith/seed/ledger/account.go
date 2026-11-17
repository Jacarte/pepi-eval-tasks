package ledger

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// Account is a general ledger account.
type Account struct {
	ID        string
	Code      string
	Name      string
	Kind      string // asset, liability, revenue or expense
	Balance   money.Cents
	Status    AccountStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AccountStatus is the lifecycle state of a account.
type AccountStatus int

const (
	AccountStatusUnknown AccountStatus = iota
	AccountStatusOpen
	AccountStatusFrozen
	AccountStatusClosed
)

var accountStatusNames = []string{"unknown", "open", "frozen", "closed"}

func (s AccountStatus) String() string {
	if s < 0 || int(s) >= len(accountStatusNames) {
		return fmt.Sprintf("AccountStatus(%d)", int(s))
	}
	return accountStatusNames[s]
}

// ParseAccountStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseAccountStatus(v string) (AccountStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range accountStatusNames {
		if i > 0 && n == v {
			return AccountStatus(i), nil
		}
	}
	return AccountStatusUnknown, fmt.Errorf("%w: unknown account status %q", ErrInvalid, v)
}

var accountTransitions = map[AccountStatus][]AccountStatus{
	AccountStatusOpen:   {AccountStatusFrozen, AccountStatusClosed},
	AccountStatusFrozen: {AccountStatusClosed},
}

// CanTransition reports whether a account may move from s to next.
func (s AccountStatus) CanTransition(next AccountStatus) bool {
	for _, t := range accountTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s AccountStatus) Terminal() bool { return len(accountTransitions[s]) == 0 }

// Validate checks the invariants every stored account must satisfy.
func (v *Account) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: account.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Code) == "" {
		return fmt.Errorf("%w: account.code is required", ErrInvalid)
	}
	if v.Balance < 0 {
		return fmt.Errorf("%w: account.balance must not be negative", ErrInvalid)
	}
	if v.Status <= AccountStatusUnknown || int(v.Status) >= len(accountStatusNames) {
		return fmt.Errorf("%w: account.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Account) Clone() *Account {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
