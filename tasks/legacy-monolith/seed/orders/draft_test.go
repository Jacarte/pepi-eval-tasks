package orders

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleDraft(n int) *Draft {
	return &Draft{
		ID:         fmt.Sprintf("draft-%03d", n),
		CustomerID: fmt.Sprintf("customerid %d;a=b,c\\d", n),
		Items:      n*27 + 69,
		Note:       fmt.Sprintf("note %d;a=b,c\\d", n),
		Total:      money.Cents(n*37 + 200),
		Status:     DraftStatusOpen,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestDraftValidate(t *testing.T) {
	if err := sampleDraft(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleDraft(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleDraft(1)
	bad.CustomerID = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing customer id: err = %v", err)
	}
	bad = sampleDraft(1)
	bad.Status = DraftStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestDraftStatusTransitions(t *testing.T) {
	for _, s := range []DraftStatus{DraftStatusOpen, DraftStatusConverted} {
		got, err := ParseDraftStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !DraftStatusOpen.CanTransition(DraftStatusAbandoned) {
		t.Fatal("first transition should be allowed")
	}
	if DraftStatusAbandoned.CanTransition(DraftStatusOpen) {
		t.Fatal("backwards transition allowed")
	}
	if !DraftStatusConverted.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeDrafts(t *testing.T) {
	items := []*Draft{sampleDraft(1), nil, sampleDraft(2), sampleDraft(3)}
	items[2].Status = DraftStatusAbandoned
	sum := SummarizeDrafts(items)
	if sum.Count != 3 || sum.ByStatus[DraftStatusOpen] != 2 || sum.ByStatus[DraftStatusAbandoned] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalItems != 369 {
		t.Errorf("TotalItems = %v, want 369", sum.TotalItems)
	}
	if sum.TotalTotal != 822 {
		t.Errorf("TotalTotal = %v, want 822", sum.TotalTotal)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopDraftsByItems(items[2:], 1); len(top) != 1 || top[0].ID != "draft-003" {
		t.Errorf("top = %v", top)
	}
}
