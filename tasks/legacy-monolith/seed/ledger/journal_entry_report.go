package ledger

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"time"

	"shopcore/money"
)

// JournalEntrySummary aggregates a batch of journal entries for the reports package.
type JournalEntrySummary struct {
	Count       int
	ByStatus    map[JournalEntryStatus]int
	TotalDebit  money.Cents
	TotalCredit money.Cents
	Earliest    time.Time
	Latest      time.Time
}

// SummarizeJournalEntries folds items into a JournalEntrySummary. Nil entries are skipped.
func SummarizeJournalEntries(items []*JournalEntry) JournalEntrySummary {
	sum := JournalEntrySummary{ByStatus: make(map[JournalEntryStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalDebit += v.Debit
		sum.TotalCredit += v.Credit
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopJournalEntriesByDebit returns the n items with the largest debit, largest first.
// Ties keep their input order.
func TopJournalEntriesByDebit(items []*JournalEntry, n int) []*JournalEntry {
	out := append([]*JournalEntry(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Debit > out[j].Debit })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupJournalEntriesByAccount buckets items by account.
func GroupJournalEntriesByAccount(items []*JournalEntry) map[string][]*JournalEntry {
	out := make(map[string][]*JournalEntry)
	for _, v := range items {
		out[v.Account] = append(out[v.Account], v)
	}
	return out
}

// JournalEntriesCreatedBetween keeps items created in [from, to).
func JournalEntriesCreatedBetween(items []*JournalEntry, from, to time.Time) []*JournalEntry {
	var out []*JournalEntry
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteJournalEntriesCSV writes items as CSV with a header row.
func WriteJournalEntriesCSV(out io.Writer, items []*JournalEntry) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "account", "debit", "credit", "memo", "reference", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Account,
			money.Cents(v.Debit).String(),
			money.Cents(v.Credit).String(),
			v.Memo,
			v.Reference,
			v.Status.String(),
			timeString(v.CreatedAt),
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DiffJournalEntry lists the fields that differ between a and b.
func DiffJournalEntry(a, b *JournalEntry) []FieldChange {
	var out []FieldChange
	if a.Account != b.Account {
		out = append(out, FieldChange{"account", fmt.Sprint(a.Account), fmt.Sprint(b.Account)})
	}
	if a.Debit != b.Debit {
		out = append(out, FieldChange{"debit", fmt.Sprint(a.Debit), fmt.Sprint(b.Debit)})
	}
	if a.Credit != b.Credit {
		out = append(out, FieldChange{"credit", fmt.Sprint(a.Credit), fmt.Sprint(b.Credit)})
	}
	if a.Memo != b.Memo {
		out = append(out, FieldChange{"memo", fmt.Sprint(a.Memo), fmt.Sprint(b.Memo)})
	}
	if a.Reference != b.Reference {
		out = append(out, FieldChange{"reference", fmt.Sprint(a.Reference), fmt.Sprint(b.Reference)})
	}
	if a.Status != b.Status {
		out = append(out, FieldChange{"status", a.Status.String(), b.Status.String()})
	}
	return out
}
