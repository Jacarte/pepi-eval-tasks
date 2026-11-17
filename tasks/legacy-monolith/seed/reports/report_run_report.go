package reports

import (
	"sort"
	"time"
)

// ReportRunSummary aggregates a batch of report runs for the reports package.
type ReportRunSummary struct {
	Count     int
	ByStatus  map[ReportRunStatus]int
	TotalRows int64
	Earliest  time.Time
	Latest    time.Time
}

// SummarizeReportRuns folds items into a ReportRunSummary. Nil entries are skipped.
func SummarizeReportRuns(items []*ReportRun) ReportRunSummary {
	sum := ReportRunSummary{ByStatus: make(map[ReportRunStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalRows += int64(v.Rows)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopReportRunsByRows returns the n items with the largest rows, largest first.
// Ties keep their input order.
func TopReportRunsByRows(items []*ReportRun, n int) []*ReportRun {
	out := append([]*ReportRun(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Rows > out[j].Rows })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupReportRunsByKind buckets items by kind.
func GroupReportRunsByKind(items []*ReportRun) map[string][]*ReportRun {
	out := make(map[string][]*ReportRun)
	for _, v := range items {
		out[v.Kind] = append(out[v.Kind], v)
	}
	return out
}

// ReportRunsCreatedBetween keeps items created in [from, to).
func ReportRunsCreatedBetween(items []*ReportRun, from, to time.Time) []*ReportRun {
	var out []*ReportRun
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
