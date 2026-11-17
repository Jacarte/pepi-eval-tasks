package scheduling

import (
	"fmt"
	"strings"
	"time"
)

// Job is a recurring background job definition.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type Job struct {
	ID        string
	Name      string
	Spec      string // schedule spec, see ParseSpec
	Owner     string
	LastRunAt time.Time
	Failures  int // consecutive failures
	Status    JobStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JobStatus is the lifecycle state of a job.
type JobStatus int

const (
	JobStatusUnknown JobStatus = iota
	JobStatusIdle
	JobStatusRunning
	JobStatusPaused
	JobStatusDisabled
)

var jobStatusNames = []string{"unknown", "idle", "running", "paused", "disabled"}

func (s JobStatus) String() string {
	if s < 0 || int(s) >= len(jobStatusNames) {
		return fmt.Sprintf("JobStatus(%d)", int(s))
	}
	return jobStatusNames[s]
}

// ParseJobStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseJobStatus(v string) (JobStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range jobStatusNames {
		if i > 0 && n == v {
			return JobStatus(i), nil
		}
	}
	return JobStatusUnknown, fmt.Errorf("%w: unknown job status %q", ErrInvalid, v)
}

var jobTransitions = map[JobStatus][]JobStatus{
	JobStatusIdle:    {JobStatusRunning, JobStatusDisabled},
	JobStatusRunning: {JobStatusPaused, JobStatusDisabled},
	JobStatusPaused:  {JobStatusDisabled},
}

// CanTransition reports whether a job may move from s to next.
func (s JobStatus) CanTransition(next JobStatus) bool {
	for _, t := range jobTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s JobStatus) Terminal() bool { return len(jobTransitions[s]) == 0 }

// Validate checks the invariants every stored job must satisfy.
func (v *Job) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return &ValidationError{Entity: "job", Field: "id", Reason: "is required"}
	}
	if strings.TrimSpace(v.Name) == "" {
		return &ValidationError{Entity: "job", Field: "name", Reason: "is required"}
	}
	if strings.TrimSpace(v.Spec) == "" {
		return &ValidationError{Entity: "job", Field: "spec", Reason: "is required"}
	}
	if v.Failures < 0 {
		return &ValidationError{Entity: "job", Field: "failures", Reason: "must not be negative"}
	}
	if v.Status <= JobStatusUnknown || int(v.Status) >= len(jobStatusNames) {
		return &ValidationError{Entity: "job", Field: "status", Reason: "is not a known status"}
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *Job) Clone() *Job {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}
