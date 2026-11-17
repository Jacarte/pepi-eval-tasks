package ledger

import (
	"encoding/csv"
	"io"
	"sort"
	"time"

	"shopcore/money"
)

// AccountSummary aggregates a batch of accounts for the reports package.
type AccountSummary struct {
	Count        int
	ByStatus     map[AccountStatus]int
	TotalBalance money.Cents
	Earliest     time.Time
	Latest       time.Time
}

// SummarizeAccounts folds items into a AccountSummary. Nil entries are skipped.
func SummarizeAccounts(items []*Account) AccountSummary {
	sum := AccountSummary{ByStatus: make(map[AccountStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalBalance += v.Balance
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopAccountsByBalance returns the n items with the largest balance, largest first.
// Ties keep their input order.
func TopAccountsByBalance(items []*Account, n int) []*Account {
	out := append([]*Account(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Balance > out[j].Balance })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupAccountsByCode buckets items by code.
func GroupAccountsByCode(items []*Account) map[string][]*Account {
	out := make(map[string][]*Account)
	for _, v := range items {
		out[v.Code] = append(out[v.Code], v)
	}
	return out
}

// AccountsCreatedBetween keeps items created in [from, to).
func AccountsCreatedBetween(items []*Account, from, to time.Time) []*Account {
	var out []*Account
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}

// WriteAccountsCSV writes items as CSV with a header row.
// FIXME: drop this once the v1 admin screens are gone.
func WriteAccountsCSV(out io.Writer, items []*Account) error {
	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "code", "name", "kind", "balance", "status", "created_at"}); err != nil {
		return err
	}
	for _, v := range items {
		rec := []string{
			v.ID,
			v.Code,
			v.Name,
			v.Kind,
			money.Cents(v.Balance).String(),
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
