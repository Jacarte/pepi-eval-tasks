package catalog

import (
	"context"
	"errors"
	"testing"

	"shopcore/audit"
	"shopcore/ctxkeys"
)

func TestPublishRetireSearch(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-cat")
	store := NewMemoryProductListingStore()
	rec := audit.NewMemory(nil)
	svc := NewService(store, rec, nil)
	_ = store.Put(ctx, &ProductListing{ID: "pl-1", SKU: "SKU-1", Title: "Trail Runner 2", Brand: "Hoka", Price: 13999, Tags: []string{"shoes"}, Status: ProductListingStatusDraft})
	_ = store.Put(ctx, &ProductListing{ID: "pl-2", SKU: "SKU-2", Title: "Free sample", Status: ProductListingStatusDraft})

	if _, err := svc.Publish(ctx, "pl-2"); !errors.Is(err, ErrNoPrice) {
		t.Fatalf("err = %v", err)
	}
	if _, err := svc.Publish(ctx, "pl-1"); err != nil {
		t.Fatal(err)
	}
	if hits, _ := svc.Search(ctx, "SHOES"); len(hits) != 1 || hits[0].ID != "pl-1" {
		t.Fatalf("hits = %v", hits)
	}
	if _, err := svc.Retire(ctx, "pl-1"); err != nil {
		t.Fatal(err)
	}
	evs := rec.Events()
	if len(evs) != 2 || evs[0].Name != "product_listing.published" || evs[1].Name != "product_listing.retired" || evs[0].RequestID != "req-cat" {
		t.Fatalf("events = %v", evs)
	}
}

func TestListingEventsCarrySKU(t *testing.T) {
	ctx := ctxkeys.WithRequestID(context.Background(), "req-sku")
	store := NewMemoryProductListingStore()
	rec := audit.NewMemory(nil)
	svc := NewService(store, rec, nil)
	_ = store.Put(ctx, &ProductListing{ID: "pl-7", SKU: "SKU-77", Title: "Mug", Price: 900, Status: ProductListingStatusDraft})
	if _, err := svc.Publish(ctx, "pl-7"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Retire(ctx, "pl-7"); err != nil {
		t.Fatal(err)
	}
	for _, ev := range rec.Events() {
		if ev.Subject != "pl-7" || ev.Attrs["sku"] != "SKU-77" {
			t.Errorf("event = %v: listing events carry the sku", ev)
		}
	}
	if _, err := svc.Reprice(ctx, "pl-7", 100); !errors.Is(err, ErrRetired) {
		t.Fatalf("reprice retired: %v", err)
	}
}
