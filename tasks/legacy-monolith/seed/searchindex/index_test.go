package searchindex

import "testing"

func TestTokenize(t *testing.T) {
	got := Tokenize("The Trail-Runner 2, for a rainy day!")
	want := []string{"trail", "runner", "rainy", "day"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}

func TestSearchRanking(t *testing.T) {
	ix := NewIndex()
	ix.Add(&Document{ID: "d1", Kind: "product", Title: "Trail runner", Body: "light shoe"})
	ix.Add(&Document{ID: "d2", Kind: "product", Title: "Road shoe", Body: "a trail capable runner", Boost: 1})
	ix.Add(&Document{ID: "d3", Kind: "page", Title: "Returns policy", Body: "how to return a shoe"})
	hits := ix.Search("trail runner", 10)
	if len(hits) != 2 || hits[0].ID != "d1" || hits[0].Score != 6 || hits[1].Score != 3 {
		t.Fatalf("hits = %v", hits)
	}
	ix.Add(&Document{ID: "d1", Kind: "product", Title: "Something else"})
	if hits := ix.Search("trail runner", 10); len(hits) != 1 || hits[0].ID != "d2" {
		t.Fatalf("after reindex = %v", hits)
	}
	ix.Remove("d2")
	if ix.Len() != 2 || len(ix.Search("trail", 0)) != 0 {
		t.Fatal("remove failed")
	}
}
