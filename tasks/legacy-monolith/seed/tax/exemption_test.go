package tax

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleExemption(n int) *Exemption {
	return &Exemption{
		ID:          fmt.Sprintf("exemption-%03d", n),
		CustomerID:  fmt.Sprintf("customerid %d;a=b,c\\d", n),
		Certificate: fmt.Sprintf("certificate-%d", n),
		Region:      fmt.Sprintf("region %d;a=b,c\\d", n),
		ExpiresAt:   time.Date(2023, time.March, n, 9, 30, 0, 0, time.UTC),
		Status:      ExemptionStatusPending,
		CreatedAt:   time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestExemptionValidate(t *testing.T) {
	if err := sampleExemption(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleExemption(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleExemption(1)
	bad.CustomerID = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing customer id: err = %v", err)
	}
	bad = sampleExemption(1)
	bad.Status = ExemptionStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestExemptionStatusTransitions(t *testing.T) {
	for _, s := range []ExemptionStatus{ExemptionStatusPending, ExemptionStatusExpired} {
		got, err := ParseExemptionStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ExemptionStatusPending.CanTransition(ExemptionStatusValid) {
		t.Fatal("first transition should be allowed")
	}
	if ExemptionStatusValid.CanTransition(ExemptionStatusPending) {
		t.Fatal("backwards transition allowed")
	}
	if !ExemptionStatusExpired.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
