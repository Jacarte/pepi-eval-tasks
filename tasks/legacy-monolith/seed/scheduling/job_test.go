package scheduling

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleJob(n int) *Job {
	return &Job{
		ID:        fmt.Sprintf("job-%03d", n),
		Name:      fmt.Sprintf("name %d;a=b,c\\d", n),
		Spec:      fmt.Sprintf("spec-%d", n),
		Owner:     fmt.Sprintf("owner %d;a=b,c\\d", n),
		LastRunAt: time.Date(2023, time.March, n, 9, 30, 0, 0, time.UTC),
		Failures:  n*32 + 281,
		Status:    JobStatusIdle,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestJobCodecRoundTrip(t *testing.T) {
	in := sampleJob(4)
	line := EncodeJob(in)
	out, err := DecodeJob(line)
	if err != nil {
		t.Fatalf("DecodeJob(%q): %v", line, err)
	}
	if got := EncodeJob(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Name != in.Name {
		t.Errorf("Name = %q, want %q", out.Name, in.Name)
	}
	if out.Spec != in.Spec {
		t.Errorf("Spec = %q, want %q", out.Spec, in.Spec)
	}
	if out.Owner != in.Owner {
		t.Errorf("Owner = %q, want %q", out.Owner, in.Owner)
	}
	if _, err := DecodeJob("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestJobValidate(t *testing.T) {
	if err := sampleJob(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleJob(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleJob(1)
	bad.Name = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing name: err = %v", err)
	}
	bad = sampleJob(1)
	bad.Status = JobStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestSummarizeJobs(t *testing.T) {
	items := []*Job{sampleJob(1), nil, sampleJob(2), sampleJob(3)}
	items[2].Status = JobStatusRunning
	sum := SummarizeJobs(items)
	if sum.Count != 3 || sum.ByStatus[JobStatusIdle] != 2 || sum.ByStatus[JobStatusRunning] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalFailures != 1035 {
		t.Errorf("TotalFailures = %v, want 1035", sum.TotalFailures)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopJobsByFailures(items[2:], 1); len(top) != 1 || top[0].ID != "job-003" {
		t.Errorf("top = %v", top)
	}
}
