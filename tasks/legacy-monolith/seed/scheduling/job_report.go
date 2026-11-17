package scheduling

import (
	"sort"
	"time"
)

// JobSummary aggregates a batch of jobs for the reports package.
type JobSummary struct {
	Count         int
	ByStatus      map[JobStatus]int
	TotalFailures int64
	Earliest      time.Time
	Latest        time.Time
}

// SummarizeJobs folds items into a JobSummary. Nil entries are skipped.
func SummarizeJobs(items []*Job) JobSummary {
	sum := JobSummary{ByStatus: make(map[JobStatus]int)}
	for _, v := range items {
		if v == nil {
			continue
		}
		sum.Count++
		sum.ByStatus[v.Status]++
		sum.TotalFailures += int64(v.Failures)
		if sum.Earliest.IsZero() || v.CreatedAt.Before(sum.Earliest) {
			sum.Earliest = v.CreatedAt
		}
		if v.CreatedAt.After(sum.Latest) {
			sum.Latest = v.CreatedAt
		}
	}
	return sum
}

// TopJobsByFailures returns the n items with the largest failures, largest first.
// Ties keep their input order.
func TopJobsByFailures(items []*Job, n int) []*Job {
	out := append([]*Job(nil), items...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Failures > out[j].Failures })
	if n >= 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// GroupJobsByName buckets items by name.
func GroupJobsByName(items []*Job) map[string][]*Job {
	out := make(map[string][]*Job)
	for _, v := range items {
		out[v.Name] = append(out[v.Name], v)
	}
	return out
}

// JobsCreatedBetween keeps items created in [from, to).
func JobsCreatedBetween(items []*Job, from, to time.Time) []*Job {
	var out []*Job
	for _, v := range items {
		if !v.CreatedAt.Before(from) && v.CreatedAt.Before(to) {
			out = append(out, v)
		}
	}
	return out
}
