package customers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleAddress(n int) *Address {
	return &Address{
		ID:         fmt.Sprintf("address-%03d", n),
		CustomerID: fmt.Sprintf("customerid %d;a=b,c\\d", n),
		Line1:      fmt.Sprintf("line1-%d", n),
		City:       fmt.Sprintf("city %d;a=b,c\\d", n),
		Postcode:   fmt.Sprintf("postcode-%d", n),
		Country:    fmt.Sprintf("country %d;a=b,c\\d", n),
		Primary:    n%2 == 0,
		Status:     AddressStatusActive,
		CreatedAt:  time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestAddressCodecRoundTrip(t *testing.T) {
	in := sampleAddress(4)
	line := EncodeAddress(in)
	out, err := DecodeAddress(line)
	if err != nil {
		t.Fatalf("DecodeAddress(%q): %v", line, err)
	}
	if got := EncodeAddress(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.CustomerID != in.CustomerID {
		t.Errorf("CustomerID = %q, want %q", out.CustomerID, in.CustomerID)
	}
	if out.Line1 != in.Line1 {
		t.Errorf("Line1 = %q, want %q", out.Line1, in.Line1)
	}
	if out.City != in.City {
		t.Errorf("City = %q, want %q", out.City, in.City)
	}
	if out.Postcode != in.Postcode {
		t.Errorf("Postcode = %q, want %q", out.Postcode, in.Postcode)
	}
	if out.Country != in.Country {
		t.Errorf("Country = %q, want %q", out.Country, in.Country)
	}
	if _, err := DecodeAddress("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryAddressStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryAddressStore()
	for n := 1; n <= 3; n++ {
		if err := st.Put(ctx, sampleAddress(n)); err != nil {
			t.Fatalf("Put(%d): %v", n, err)
		}
	}
	got, err := st.Get(ctx, "address-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = AddressStatusArchived
	again, _ := st.Get(ctx, "address-002")
	if again.Status != AddressStatusActive {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Put(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.List(ctx, AddressFilter{Status: AddressStatusActive})
	if err != nil || len(list) != 2 || list[0].ID != "address-001" {
		t.Fatalf("List by status = %v, %v", list, err)
	}
	if err := st.Delete(ctx, "address-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, "address-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestSummarizeAddresses(t *testing.T) {
	items := []*Address{sampleAddress(1), nil, sampleAddress(2), sampleAddress(3)}
	items[2].Status = AddressStatusArchived
	sum := SummarizeAddresses(items)
	if sum.Count != 3 || sum.ByStatus[AddressStatusActive] != 2 || sum.ByStatus[AddressStatusArchived] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
}
