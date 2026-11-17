package customers

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleCustomer(n int) *Customer {
	return &Customer{
		ID:            fmt.Sprintf("customer-%03d", n),
		Email:         fmt.Sprintf("email %d;a=b,c\\d", n),
		DisplayName:   fmt.Sprintf("displayname-%d", n),
		Segment:       fmt.Sprintf("segment %d;a=b,c\\d", n),
		LifetimeValue: money.Cents(n*22 + 216),
		Marketing:     n%2 == 0,
		Status:        CustomerStatusPending,
		CreatedAt:     time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestCustomerCodecRoundTrip(t *testing.T) {
	in := sampleCustomer(4)
	line := EncodeCustomer(in)
	out, err := DecodeCustomer(line)
	if err != nil {
		t.Fatalf("DecodeCustomer(%q): %v", line, err)
	}
	if got := EncodeCustomer(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Email != in.Email {
		t.Errorf("Email = %q, want %q", out.Email, in.Email)
	}
	if out.DisplayName != in.DisplayName {
		t.Errorf("DisplayName = %q, want %q", out.DisplayName, in.DisplayName)
	}
	if out.Segment != in.Segment {
		t.Errorf("Segment = %q, want %q", out.Segment, in.Segment)
	}
	if _, err := DecodeCustomer("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestCustomerValidate(t *testing.T) {
	if err := sampleCustomer(1).Validate(); err != nil {
		t.Fatalf("valid sample rejected: %v", err)
	}
	bad := sampleCustomer(1)
	bad.ID = " "
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank id: err = %v", err)
	}
	bad = sampleCustomer(1)
	bad.Email = ""
	if err := bad.Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing email: err = %v", err)
	}
	bad = sampleCustomer(1)
	bad.Status = CustomerStatusUnknown
	if err := bad.Validate(); err == nil {
		t.Fatal("unknown status accepted")
	}
}

func TestSummarizeCustomers(t *testing.T) {
	items := []*Customer{sampleCustomer(1), nil, sampleCustomer(2), sampleCustomer(3)}
	items[2].Status = CustomerStatusActive
	sum := SummarizeCustomers(items)
	if sum.Count != 3 || sum.ByStatus[CustomerStatusPending] != 2 || sum.ByStatus[CustomerStatusActive] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalLifetimeValue != 780 {
		t.Errorf("TotalLifetimeValue = %v, want 780", sum.TotalLifetimeValue)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopCustomersByLifetimeValue(items[2:], 1); len(top) != 1 || top[0].ID != "customer-003" {
		t.Errorf("top = %v", top)
	}
}
