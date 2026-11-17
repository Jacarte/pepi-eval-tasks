package ledger

import (
	"context"
	"errors"
	"strings"
)

// ErrEmptyMemo is returned by Amend for a blank memo.
var ErrEmptyMemo = errors.New("ledger: memo is required")

// Amend replaces the memo of a draft entry. Posted entries are immutable.
func (s *Service) Amend(ctx context.Context, entryID, memo string) (*JournalEntry, error) {
	if strings.TrimSpace(memo) == "" {
		return nil, ErrEmptyMemo
	}
	e, err := s.entries.Get(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if e.Status != JournalEntryStatusDraft {
		return nil, ErrNotDraft
	}
	e.Memo = strings.TrimSpace(memo)
	e.UpdatedAt = s.clock.Now()
	if err := s.entries.Put(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}
