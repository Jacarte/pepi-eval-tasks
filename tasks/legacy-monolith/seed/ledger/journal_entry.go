package ledger

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// JournalEntry is a double-entry journal line pair.
type JournalEntry struct {
	ID        string
	Account   string // account code
	Debit     money.Cents
	Credit    money.Cents
	Memo      string
	Reference string // external reference, e.g. an order id
	Status    JournalEntryStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JournalEntryStatus is the lifecycle state of a journal entry.
type JournalEntryStatus int

const (
	JournalEntryStatusUnknown JournalEntryStatus = iota
	JournalEntryStatusDraft
	JournalEntryStatusPosted
	JournalEntryStatusReversed
)

var journalEntryStatusNames = []string{"unknown", "draft", "posted", "reversed"}

func (s JournalEntryStatus) String() string {
	if s < 0 || int(s) >= len(journalEntryStatusNames) {
		return fmt.Sprintf("JournalEntryStatus(%d)", int(s))
	}
	return journalEntryStatusNames[s]
}

// ParseJournalEntryStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseJournalEntryStatus(v string) (JournalEntryStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range journalEntryStatusNames {
		if i > 0 && n == v {
			return JournalEntryStatus(i), nil
		}
	}
	return JournalEntryStatusUnknown, fmt.Errorf("%w: unknown journal entry status %q", ErrInvalid, v)
}

var journalEntryTransitions = map[JournalEntryStatus][]JournalEntryStatus{
	JournalEntryStatusDraft:  {JournalEntryStatusPosted, JournalEntryStatusReversed},
	JournalEntryStatusPosted: {JournalEntryStatusReversed},
}

// CanTransition reports whether a journal entry may move from s to next.
func (s JournalEntryStatus) CanTransition(next JournalEntryStatus) bool {
	for _, t := range journalEntryTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s JournalEntryStatus) Terminal() bool { return len(journalEntryTransitions[s]) == 0 }

// Validate checks the invariants every stored journal entry must satisfy.
func (v *JournalEntry) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: journal_entry.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Account) == "" {
		return fmt.Errorf("%w: journal_entry.account is required", ErrInvalid)
	}
	if v.Debit < 0 {
		return fmt.Errorf("%w: journal_entry.debit must not be negative", ErrInvalid)
	}
	if v.Credit < 0 {
		return fmt.Errorf("%w: journal_entry.credit must not be negative", ErrInvalid)
	}
	if v.Status <= JournalEntryStatusUnknown || int(v.Status) >= len(journalEntryStatusNames) {
		return fmt.Errorf("%w: journal_entry.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *JournalEntry) Clone() *JournalEntry {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
