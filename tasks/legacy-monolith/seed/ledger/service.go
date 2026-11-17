// Package ledger is the general ledger: journal entries and account balances.
package ledger

import (
	"context"
	"errors"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

var (
	ErrUnbalanced = errors.New("ledger: entry must have exactly one of debit or credit")
	ErrNotDraft   = errors.New("ledger: entry is not a draft")
	ErrNotPosted  = errors.New("ledger: entry is not posted")
	ErrFrozen     = errors.New("ledger: account is not open")
)

type Service struct {
	entries  JournalEntryStore
	accounts AccountStore
	audit    audit.Recorder
	clock    clock.Clock
}

func NewService(entries JournalEntryStore, accounts AccountStore, rec audit.Recorder, clk clock.Clock) *Service {
	return &Service{entries: entries, accounts: accounts, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk)}
}

// Post books a draft entry against its account.
func (s *Service) Post(ctx context.Context, entryID string) (*JournalEntry, error) {
	const op = "ledger.post"
	e, err := s.entries.Get(ctx, entryID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if e.Status != JournalEntryStatusDraft {
		return nil, errs.Wrap(op, ErrNotDraft)
	}
	if (e.Debit == 0) == (e.Credit == 0) {
		return nil, errs.Wrap(op, ErrUnbalanced)
	}
	acct, err := s.accounts.Get(ctx, e.Account)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if acct.Status != AccountStatusOpen {
		return nil, errs.Wrap(op, ErrFrozen)
	}
	acct.Balance += e.Debit - e.Credit
	e.Status = JournalEntryStatusPosted
	e.UpdatedAt = s.clock.Now()
	// The account write and the entry write are not atomic; the nightly
	// reconciliation job repairs the (rare) half-applied case.
	if err := s.accounts.Put(ctx, acct); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.entries.Put(ctx, e); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, audit.Event{Name: "journal_entry.posted", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: e.ID, Attrs: map[string]string{"account": e.Account}}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return e, nil
}

// Reverse undoes a posted entry.
func (s *Service) Reverse(ctx context.Context, entryID string) (*JournalEntry, error) {
	const op = "ledger.reverse"
	e, err := s.entries.Get(ctx, entryID)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if e.Status != JournalEntryStatusPosted {
		return nil, errs.Wrap(op, ErrNotPosted)
	}
	acct, err := s.accounts.Get(ctx, e.Account)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	acct.Balance -= e.Debit - e.Credit
	e.Status = JournalEntryStatusReversed
	e.UpdatedAt = s.clock.Now()
	if err := s.accounts.Put(ctx, acct); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.entries.Put(ctx, e); err != nil {
		return nil, errs.Wrap(op, err)
	}
	if err := s.audit.Record(ctx, audit.Event{Name: "journal_entry.reversed", RequestID: ctxkeys.RequestIDFrom(ctx), Subject: e.ID, Attrs: map[string]string{"account": e.Account}}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return e, nil
}

// TrialBalance returns the sum of all account balances; it should be zero.
func (s *Service) TrialBalance(ctx context.Context) (int64, error) {
	const op = "ledger.trialbalance"
	all, err := s.accounts.List(ctx, AccountFilter{})
	if err != nil {
		return 0, errs.Wrap(op, err)
	}
	var total int64
	for _, a := range all {
		total += int64(a.Balance)
	}
	return total, nil
}
