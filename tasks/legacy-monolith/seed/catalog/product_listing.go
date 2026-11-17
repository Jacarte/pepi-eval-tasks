package catalog

import (
	"fmt"
	"strings"
	"time"

	"shopcore/money"
)

// ProductListing is the merchandised view of a SKU.
//
// The zero value is not valid; build one with all required fields and run
// Validate before handing it to a store.
type ProductListing struct {
	ID        string
	SKU       string
	Title     string
	Brand     string
	Price     money.Cents // list price
	Tags      []string    // free-form merchandising tags
	Featured  bool
	Status    ProductListingStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ProductListingStatus is the lifecycle state of a product listing.
type ProductListingStatus int

const (
	ProductListingStatusUnknown ProductListingStatus = iota
	ProductListingStatusDraft
	ProductListingStatusPublished
	ProductListingStatusRetired
)

var productListingStatusNames = []string{"unknown", "draft", "published", "retired"}

func (s ProductListingStatus) String() string {
	if s < 0 || int(s) >= len(productListingStatusNames) {
		return fmt.Sprintf("ProductListingStatus(%d)", int(s))
	}
	return productListingStatusNames[s]
}

// ParseProductListingStatus parses the wire name of a status. Matching is
// case-insensitive because the old importer wrote upper-case values.
func ParseProductListingStatus(v string) (ProductListingStatus, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	for i, n := range productListingStatusNames {
		if i > 0 && n == v {
			return ProductListingStatus(i), nil
		}
	}
	return ProductListingStatusUnknown, fmt.Errorf("%w: unknown product listing status %q", ErrInvalid, v)
}

var productListingTransitions = map[ProductListingStatus][]ProductListingStatus{
	ProductListingStatusDraft:     {ProductListingStatusPublished, ProductListingStatusRetired},
	ProductListingStatusPublished: {ProductListingStatusRetired},
}

// CanTransition reports whether a product listing may move from s to next.
func (s ProductListingStatus) CanTransition(next ProductListingStatus) bool {
	for _, t := range productListingTransitions[s] {
		if t == next {
			return true
		}
	}
	return false
}

// Terminal reports whether no further transitions are possible.
func (s ProductListingStatus) Terminal() bool { return len(productListingTransitions[s]) == 0 }

// Validate checks the invariants every stored product listing must satisfy.
func (v *ProductListing) Validate() error {
	if strings.TrimSpace(v.ID) == "" {
		return fmt.Errorf("%w: product_listing.id is required", ErrInvalid)
	}
	if strings.TrimSpace(v.SKU) == "" {
		return fmt.Errorf("%w: product_listing.sku is required", ErrInvalid)
	}
	if strings.TrimSpace(v.Title) == "" {
		return fmt.Errorf("%w: product_listing.title is required", ErrInvalid)
	}
	if v.Price < 0 {
		return fmt.Errorf("%w: product_listing.price must not be negative", ErrInvalid)
	}
	for _, tag := range v.Tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("%w: product_listing.tags contains an empty tag", ErrInvalid)
		}
	}
	if v.Status <= ProductListingStatusUnknown || int(v.Status) >= len(productListingStatusNames) {
		return fmt.Errorf("%w: product_listing.status is not a known status", ErrInvalid)
	}
	return nil
}

// Clone returns a deep copy so callers can mutate the result freely.
func (v *ProductListing) Clone() *ProductListing {
	if v == nil {
		return nil
	}
	cp := *v
	if v.Tags != nil {
		cp.Tags = append([]string(nil), v.Tags...)
	}
	return &cp
}
