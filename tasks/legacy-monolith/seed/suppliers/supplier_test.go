package suppliers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func sampleSupplier(n int) *Supplier {
	return &Supplier{
		ID:           fmt.Sprintf("supplier-%03d", n),
		Name:         fmt.Sprintf("name %d;a=b,c\\d", n),
		Country:      fmt.Sprintf("country-%d", n),
		ContactEmail: fmt.Sprintf("contactemail %d;a=b,c\\d", n),
		LeadTimeDays: n*8 + 115,
		Preferred:    n%2 == 0,
		Status:       SupplierStatusProspect,
		CreatedAt:    time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestSupplierCodecRoundTrip(t *testing.T) {
	in := sampleSupplier(4)
	line := EncodeSupplier(in)
	out, err := DecodeSupplier(line)
	if err != nil {
		t.Fatalf("DecodeSupplier(%q): %v", line, err)
	}
	if got := EncodeSupplier(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Name != in.Name {
		t.Errorf("Name = %q, want %q", out.Name, in.Name)
	}
	if out.Country != in.Country {
		t.Errorf("Country = %q, want %q", out.Country, in.Country)
	}
	if out.ContactEmail != in.ContactEmail {
		t.Errorf("ContactEmail = %q, want %q", out.ContactEmail, in.ContactEmail)
	}
	if _, err := DecodeSupplier("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemorySupplierStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemorySupplierStore()
	for n := 1; n <= 3; n++ {
		if err := st.Save(ctx, sampleSupplier(n)); err != nil {
			t.Fatalf("Save(%d): %v", n, err)
		}
	}
	got, err := st.Find(ctx, "supplier-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = SupplierStatusSuspended
	again, _ := st.Find(ctx, "supplier-002")
	if again.Status != SupplierStatusProspect {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Save(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.All(ctx, SupplierFilter{Status: SupplierStatusProspect})
	if err != nil || len(list) != 2 || list[0].ID != "supplier-001" {
		t.Fatalf("All by status = %v, %v", list, err)
	}
	if err := st.Remove(ctx, "supplier-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Find(ctx, "supplier-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestSupplierStatusTransitions(t *testing.T) {
	for _, s := range []SupplierStatus{SupplierStatusProspect, SupplierStatusSuspended} {
		got, err := ParseSupplierStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !SupplierStatusProspect.CanTransition(SupplierStatusOnboarded) {
		t.Fatal("first transition should be allowed")
	}
	if SupplierStatusOnboarded.CanTransition(SupplierStatusProspect) {
		t.Fatal("backwards transition allowed")
	}
	if !SupplierStatusSuspended.Terminal() {
		t.Fatal("last status should be terminal")
	}
}

func TestSummarizeSuppliers(t *testing.T) {
	items := []*Supplier{sampleSupplier(1), nil, sampleSupplier(2), sampleSupplier(3)}
	items[2].Status = SupplierStatusOnboarded
	sum := SummarizeSuppliers(items)
	if sum.Count != 3 || sum.ByStatus[SupplierStatusProspect] != 2 || sum.ByStatus[SupplierStatusOnboarded] != 1 {
		t.Fatalf("counts = %+v", sum)
	}
	if sum.TotalLeadTimeDays != 393 {
		t.Errorf("TotalLeadTimeDays = %v, want 393", sum.TotalLeadTimeDays)
	}
	if !sum.Earliest.Equal(items[0].CreatedAt) || !sum.Latest.Equal(items[3].CreatedAt) {
		t.Errorf("range = %v..%v", sum.Earliest, sum.Latest)
	}
	if top := TopSuppliersByLeadTimeDays(items[2:], 1); len(top) != 1 || top[0].ID != "supplier-003" {
		t.Errorf("top = %v", top)
	}
}

func TestWriteSuppliersCSV(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteSuppliersCSV(&buf, []*Supplier{sampleSupplier(1), sampleSupplier(2)}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "id,") {
		t.Fatalf("csv = %q", buf.String())
	}
}
