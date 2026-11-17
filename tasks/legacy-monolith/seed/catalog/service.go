// Package catalog manages product listings and navigation categories.
package catalog

import (
	"context"
	"errors"
	"strings"

	"shopcore/audit"
	"shopcore/clock"
	"shopcore/ctxkeys"
	"shopcore/errs"
)

var (
	ErrNotDraft     = errors.New("catalog: listing is not a draft")
	ErrNotPublished = errors.New("catalog: listing is not published")
	ErrNoPrice      = errors.New("catalog: listing has no price")
)

// Service is the merchandising back office.
type Service struct {
	listings ProductListingStore
	audit    audit.Recorder
	clock    clock.Clock
}

func NewService(listings ProductListingStore, rec audit.Recorder, clk clock.Clock) *Service {
	return &Service{listings: listings, audit: audit.OrDiscard(rec), clock: clock.OrSystem(clk)}
}

// Publish makes a draft listing visible on the storefront.
func (s *Service) Publish(ctx context.Context, id string) (*ProductListing, error) {
	const op = "catalog.publish"
	l, err := s.listings.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if l.Status != ProductListingStatusDraft {
		return nil, errs.Wrap(op, ErrNotDraft)
	}
	if l.Price <= 0 {
		return nil, errs.Wrap(op, ErrNoPrice)
	}
	l.Status = ProductListingStatusPublished
	l.UpdatedAt = s.clock.Now()
	if err := s.listings.Put(ctx, l); err != nil {
		return nil, errs.Wrap(op, err)
	}
	reqID := ctxkeys.RequestIDFrom(ctx)
	if err := s.audit.Record(ctx, audit.Event{Name: "product_listing.published", RequestID: reqID, Subject: l.ID, Attrs: map[string]string{"sku": l.SKU}}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return l, nil
}

// Retire removes a published listing from the storefront.
func (s *Service) Retire(ctx context.Context, id string) (*ProductListing, error) {
	const op = "catalog.retire"
	l, err := s.listings.Get(ctx, id)
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	if l.Status != ProductListingStatusPublished {
		return nil, errs.Wrap(op, ErrNotPublished)
	}
	l.Status = ProductListingStatusRetired
	l.Featured = false
	l.UpdatedAt = s.clock.Now()
	if err := s.listings.Put(ctx, l); err != nil {
		return nil, errs.Wrap(op, err)
	}
	reqID := ctxkeys.RequestIDFrom(ctx)
	if err := s.audit.Record(ctx, audit.Event{Name: "product_listing.retired", RequestID: reqID, Subject: l.ID, Attrs: map[string]string{"sku": l.SKU}}); err != nil {
		return nil, errs.Wrap(op, err)
	}
	return l, nil
}

// Search does a naive title/brand/tag match over published listings. The
// storefront uses searchindex; this is for the back office only.
func (s *Service) Search(ctx context.Context, q string) ([]*ProductListing, error) {
	const op = "catalog.search"
	all, err := s.listings.List(ctx, ProductListingFilter{Status: ProductListingStatusPublished})
	if err != nil {
		return nil, errs.Wrap(op, err)
	}
	q = strings.ToLower(strings.TrimSpace(q))
	var out []*ProductListing
	for _, l := range all {
		hay := strings.ToLower(l.Title + " " + l.Brand + " " + strings.Join(l.Tags, " "))
		if q == "" || strings.Contains(hay, q) {
			out = append(out, l)
		}
	}
	return out, nil
}
