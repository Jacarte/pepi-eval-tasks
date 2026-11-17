package catalog

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"shopcore/money"
)

func sampleProductListing(n int) *ProductListing {
	return &ProductListing{
		ID:        fmt.Sprintf("product_listing-%03d", n),
		SKU:       fmt.Sprintf("sku %d;a=b,c\\d", n),
		Title:     fmt.Sprintf("title-%d", n),
		Brand:     fmt.Sprintf("brand %d;a=b,c\\d", n),
		Price:     money.Cents(n*12 + 289),
		Tags:      []string{"x,y", fmt.Sprintf("t%d", n)},
		Featured:  n%2 == 0,
		Status:    ProductListingStatusDraft,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestProductListingCodecRoundTrip(t *testing.T) {
	in := sampleProductListing(4)
	line := EncodeProductListing(in)
	out, err := DecodeProductListing(line)
	if err != nil {
		t.Fatalf("DecodeProductListing(%q): %v", line, err)
	}
	if got := EncodeProductListing(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.SKU != in.SKU {
		t.Errorf("SKU = %q, want %q", out.SKU, in.SKU)
	}
	if out.Title != in.Title {
		t.Errorf("Title = %q, want %q", out.Title, in.Title)
	}
	if out.Brand != in.Brand {
		t.Errorf("Brand = %q, want %q", out.Brand, in.Brand)
	}
	if _, err := DecodeProductListing("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestProductListingStatusTransitions(t *testing.T) {
	for _, s := range []ProductListingStatus{ProductListingStatusDraft, ProductListingStatusRetired} {
		got, err := ParseProductListingStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !ProductListingStatusDraft.CanTransition(ProductListingStatusPublished) {
		t.Fatal("first transition should be allowed")
	}
	if ProductListingStatusPublished.CanTransition(ProductListingStatusDraft) {
		t.Fatal("backwards transition allowed")
	}
	if !ProductListingStatusRetired.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
