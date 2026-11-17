package reports

import (
	"fmt"
	"strings"
	"time"
)

// ReportRun is one execution of a scheduled report.
type ReportRun struct {
	ID          string
	Kind        string
	RequestedBy string
	Rows        int
	Format      string // csv or json
	Status      ReportRunStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ReportRunStatus is the lifecycle state of a report run.
type ReportRunStatus int

const (
	ReportRunStatusUnknown ReportRunStatus = iota
	ReportRunStatusQueued
	ReportRunStatusRunning
	ReportRunStatusDone
	ReportRunStatusFailed
)

var reportRunStatusNames = []string{"unknown", "queued", "running", "done", "failed"}

func (s ReportRunStatus) String() string {
	if s < 0 || int(s) >= len(reportRunStatusNames) {
		return fmt.Sprintf("ReportRunStatus(%d)", int(s))
	}
	return reportRunStatusNames[s]
}

// ParseReportRunStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseReportRunStatus(v string) (ReportRunStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range reportRunStatusNames {
		if i > 0 && n == v {
			return ReportRunStatus(i), nil
		}
	}
	return ReportRunStatusUnknown, fmt.Errorf("%w: unknown report run status %q", ErrInvalid, v)
}

var reportRunTransitions = map[ReportRunStatus][]ReportRunStatus{
	ReportRunStatusQueued:  {ReportRunStatusRunning, ReportRunStatusFailed},
	ReportRunStatusRunning: {ReportRunStatusDone, ReportRunStatusFailed},
	ReportRunStatusDone:    {ReportRunStatusFailed},
}

// CanTransition reports whether a report run may move from s to next.
func (s ReportRunStatus) CanTransition(next ReportRunStatus) bool {
	for _, t := range reportRunTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ReportRunStatus) Terminal() bool { return len(reportRunTransitions[s]) == 0 }

// Validate checks the invariants every stored report run must satisfy.
func (v *ReportRun) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: report_run.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Kind) == "" {
		return fmt.Errorf("%w: report_run.kind is required", ErrInvalid)
	}
	if v.Rows < 0 {
		return fmt.Errorf("%w: report_run.rows must not be negative", ErrInvalid)
	}
	if v.Status <= ReportRunStatusUnknown || int(v.Status) >= len(reportRunStatusNames) {
		return fmt.Errorf("%w: report_run.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *ReportRun) Clone() *ReportRun {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
