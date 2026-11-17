package ledger

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

func TestPostReverse(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-gl")
	entries, accounts := NewMemoryJournalEntryStore(), NewMemoryAccountStore()
	rec := audit.NewMemory(nil)
	svc := NewService(entries, accounts, rec, nil)
	_ = accounts.Put(ctx, &Account{ID: "1000", Code: "1000", Name: "Cash", Kind: "asset", Status: AccountStatusOpen})
	_ = accounts.Put(ctx, &Account{ID: "4000", Code: "4000", Name: "Sales", Kind: "revenue", Status: AccountStatusOpen, Balance: 500})
	_ = entries.Put(ctx, &JournalEntry{ID: "je-1", Account: "1000", Debit: 500, Status: JournalEntryStatusDraft})
	_ = entries.Put(ctx, &JournalEntry{ID: "je-2", Account: "1000", Debit: 5, Credit: 5, Status: JournalEntryStatusDraft})

	if _, err := svc.Post(ctx, "je-2"); !errors.Is(err, ErrUnbalanced) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Post(ctx, "je-1"); err != nil {
		t.Fatal(err)
	}
	cash, _ := accounts.Get(ctx, "1000")
	if cash.Balance != 500 {
		t.Fatalf("cash = %v", cash.Balance)
	}
	if _, err := svc.Reverse(ctx, "je-1"); err != nil {
		t.Fatal(err)
	}
	if tb, _ := svc.TrialBalance(ctx); tb != 500 {
		t.Fatalf("trial balance = %d", tb)
	}
	evs := rec.Events()
	if len(evs) != 2 || evs[0].Name != "journal_entry.posted" || evs[1].Name != "journal_entry.reversed" {
		t.Fatalf("events = %v", evs)
	}
}

func TestEntryEventsCarryAccount(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-acc")
	entries, accounts := NewMemoryJournalEntryStore(), NewMemoryAccountStore()
	rec := audit.NewMemory(nil)
	svc := NewService(entries, accounts, rec, nil)
	_ = accounts.Put(ctx, &Account{ID: "2000", Code: "2000", Name: "Payables", Kind: "liability", Status: AccountStatusOpen})
	_ = entries.Put(ctx, &JournalEntry{ID: "je-9", Account: "2000", Debit: 40, Status: JournalEntryStatusDraft})
	if _, err := svc.Amend(ctx, "je-9", "  "); !errors.Is(err, ErrEmptyMemo) {
		t.Fatalf("blank memo: %v", err)
	}
	if _, err := svc.Post(ctx, "je-9"); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	if len(evs) != 1 || evs[0].Subject != "je-9" || evs[0].Attrs["account"] != "2000" {
		t.Fatalf("events = %v: entry events carry the account code", evs)
	}
	if _, err := svc.Amend(ctx, "je-9", "late memo"); !errors.Is(err, ErrNotDraft) {
		t.Fatalf("amend posted: %v", err)
	}
}
