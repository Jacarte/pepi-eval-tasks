package reports

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleReportRun(n int) *ReportRun {
	return &ReportRun{
		ID:          fmt.Sprintf("report_run-%03d", n),
		Kind:        fmt.Sprintf("kind %d;a=b,c\\d", n),
		RequestedBy: fmt.Sprintf("requestedby-%d", n),
		Rows:        n*33 + 467,
		Format:      fmt.Sprintf("format-%d", n),
		Status:      ReportRunStatusQueued,
		CreatedAt:   time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestMemoryReportRunStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryReportRunStore()
	for n := 1; n <= 3; n++ {
		if err := st.Save(ctx, sampleReportRun(n)); err != nil {
			t.Fatalf("Save(%d): %v", n, err)
		}
	}
	got, err := st.Find(ctx, "report_run-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = ReportRunStatusFailed
	again, _ := st.Find(ctx, "report_run-002")
	if again.Status != ReportRunStatusQueued {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Save(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.All(ctx, ReportRunFilter{Status: ReportRunStatusQueued})
	if err != nil || len(list) != 2 || list[0].ID != "report_run-001" {
		t.Fatalf("All by status = %v, %v", list, err)
	}
	if err := st.Remove(ctx, "report_run-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Find(ctx, "report_run-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestReportRunValidate(t *testing.T) {
	if err := sampleReportRun(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleReportRun(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleReportRun(1)
	bad.Kind = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing kind: err = %v", err)
	}
	bad = sampleReportRun(1)
	bad.Status = ReportRunStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestReportRunStatusTransitions(t *testing.T) {
	for _, s := range []ReportRunStatus{ReportRunStatusQueued, ReportRunStatusFailed} {
		got, err := ParseReportRunStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ReportRunStatusQueued.CanTransition(ReportRunStatusRunning) {
		t.Fatal("first transition should be allowed")
	}
	if ReportRunStatusRunning.CanTransition(ReportRunStatusQueued) {
		t.Fatal("backwards transition allowed")
	}
	if !ReportRunStatusFailed.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
