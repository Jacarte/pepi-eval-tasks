package searchindex

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func sampleDocument(n int) *Document {
	return &Document{
		ID:        fmt.Sprintf("document-%03d", n),
		Kind:      fmt.Sprintf("kind %d;a=b,c\\d", n),
		Title:     fmt.Sprintf("title-%d", n),
		Body:      fmt.Sprintf("body %d;a=b,c\\d", n),
		Boost:     n*10 + 263,
		Tags:      []string{"x,y", fmt.Sprintf("t%d", n)},
		Status:    DocumentStatusPending,
		CreatedAt: time.Date(2023, time.January, n, 8, 0, 0, 0, time.UTC),
	}
}

func TestDocumentCodecRoundTrip(t *testing.T) {
	in := sampleDocument(4)
	line := EncodeDocument(in)
	out, err := DecodeDocument(line)
	if err != nil {
		t.Fatalf("DecodeDocument(%q): %v", line, err)
	}
	if got := EncodeDocument(out); got != line {
		t.Fatalf("round trip mismatch:\n got %s\nwant %s", got, line)
	}
	if out.Kind != in.Kind {
		t.Errorf("Kind = %q, want %q", out.Kind, in.Kind)
	}
	if out.Title != in.Title {
		t.Errorf("Title = %q, want %q", out.Title, in.Title)
	}
	if out.Body != in.Body {
		t.Errorf("Body = %q, want %q", out.Body, in.Body)
	}
	if _, err := DecodeDocument("id"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed input: err = %v, want ErrInvalid", err)
	}
}

func TestMemoryDocumentStore(t *testing.T) {
	ctx := context.Background()
	st := NewMemoryDocumentStore()
	for n := 1; n <= 3; n++ {
		if err := st.Put(ctx, sampleDocument(n)); err != nil {
			t.Fatalf("Put(%d): %v", n, err)
		}
	}
	got, err := st.Get(ctx, "document-002")
	if err != nil {
		t.Fatal(err)
	}
	got.Status = DocumentStatusStale
	again, _ := st.Get(ctx, "document-002")
	if again.Status != DocumentStatusPending {
		t.Fatal("store handed out a shared pointer")
	}
	if err := st.Put(ctx, got); err != nil {
		t.Fatal(err)
	}
	list, err := st.List(ctx, DocumentFilter{Status: DocumentStatusPending})
	if err != nil || len(list) != 2 || list[0].ID != "document-001" {
		t.Fatalf("List by status = %v, %v", list, err)
	}
	if err := st.Delete(ctx, "document-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Get(ctx, "document-001"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v, want ErrNotFound", err)
	}
	if st.Len() != 2 {
		t.Fatalf("Len = %d, want 2", st.Len())
	}
}

func TestDocumentStatusTransitions(t *testing.T) {
	for _, s := range []DocumentStatus{DocumentStatusPending, DocumentStatusStale} {
		got, err := ParseDocumentStatus(s.String())
		if err != nil || got != s {
			t.Fatalf("Parse(%q) = %v, %v", s.String(), got, err)
		}
	}
	if !DocumentStatusPending.CanTransition(DocumentStatusIndexed) {
		t.Fatal("first transition should be allowed")
	}
	if DocumentStatusIndexed.CanTransition(DocumentStatusPending) {
		t.Fatal("backwards transition allowed")
	}
	if !DocumentStatusStale.Terminal() {
		t.Fatal("last status should be terminal")
	}
}
