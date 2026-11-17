package ledger

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleJournalEntry(n int) *JournalEntry {
	return &JournalEntry{
		ID:        fmt.Sprintf("journal_entry-%03d", n),
		Account:   fmt.Sprintf("account %d;a=b,c\\d", n),
		Debit:     money.Cents(n*32 + 314),
		Credit:    money.Cents(n*15 + 400),
		Memo:      fmt.Sprintf("memo-%d", n),
		Reference: fmt.Sprintf("reference %d;a=b,c\\d", n),
		Status:    JournalEntryStatusDraft,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestJournalEntryCodecRoundTrip(t *testing.T) {
	in := sampleJournalEntry(4)
	line := EncodeJournalEntry(in)
	out, err := DecodeJournalEntry(line)
	if err != nil {
		t.Fatalf("DecodeJournalEntry(%q): %v", line, err)
	}
	if got := EncodeJournalEntry(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Account != in.Account {
		t.Errorf("Account = %q, want %q", out.Account, in.Account)
	}
	if out.Memo != in.Memo {
		t.Errorf("Memo = %q, want %q", out.Memo, in.Memo)
	}
	if out.Reference != in.Reference {
		t.Errorf("Reference = %q, want %q", out.Reference, in.Reference)
	}
	if _, err := DecodeJournalEntry("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestJournalEntryValidate(t *testing.T) {
	if err := sampleJournalEntry(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleJournalEntry(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleJournalEntry(1)
	bad.Account = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing account: err = %v", err)
	}
	bad = sampleJournalEntry(1)
	bad.Status = JournalEntryStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestJournalEntryStatusTransitions(t *testing.T) {
	for _, s := range []JournalEntryStatus{JournalEntryStatusDraft, JournalEntryStatusReversed} {
		got, err := ParseJournalEntryStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !JournalEntryStatusDraft.CanTransition(JournalEntryStatusPosted) {
		t.Fatal("first transition should be allowed")
	}
	if JournalEntryStatusPosted.CanTransition(JournalEntryStatusDraft) {
		t.Fatal("backwards transition allowed")
	}
	if !JournalEntryStatusReversed.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeJournalEntries(t *testing.T) {
	items := []*JournalEntry{sampleJournalEntry(1), nil, sampleJournalEntry(2), sampleJournalEntry(3)}
	items[2].Status = JournalEntryStatusPosted
	sum := SummarizeJournalEntries(items)
	if sum.Count != 3 || sum.ByStatus[JournalEntryStatusDraft] != 2 || sum.ByStatus[JournalEntryStatusPosted] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalDebit != 1134 {
		t.Errorf("TotalDebit = %v, want 1134", sum.TotalDebit)
	}
	if sum.TotalCredit != 1290 {
		t.Errorf("TotalCredit = %v, want 1290", sum.TotalCredit)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopJournalEntriesByDebit(items[2:], 1); len(top) != 1 || top[0].ID != "journal_entry-003" {
		t.Errorf("top = %v", top)
	}
}
