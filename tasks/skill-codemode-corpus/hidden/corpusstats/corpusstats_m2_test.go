package corpusstats

import (
	"testing"
)

func TestM2LinesMatchesReferenceOnCorpus(t *testing.T) {
	want := map[string]int{}
	for _, f := range refWalk(t, m1Corpus) {
		want[f.ext] += refLines(f.data)
	}
	got, err := Lines(m1Corpus)
	if err != nil {
		t.Fatal(err)
	}
	if !sameCounts(got, want) {
		t.Fatalf("Lines(corpus) = %v, want %v", got, want)
	}
}

func TestM2LinesFixture(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"a.txt":   "one\r\ntwo",         // CRLF is one break; no final newline still counts: 2
		"d.txt":   "x\ny\r\n",           // 2
		"b.md":    "",                   // empty: 0
		"c.go":    "\n\n",               // 2 empty lines
		"e.go":    "lone\rcr",           // a lone CR is not a break: 1
		"f.go":    "   \n\t\n",          // whitespace-only lines count: 2
		"x.json":  "not counted\n\n\n",  // ignored
		"g/h.txt": "last\nline without", // 2
	})
	got, err := Lines(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{".txt": 6, ".md": 0, ".go": 5}
	if !sameCounts(got, want) {
		t.Fatalf("Lines(fixture) = %v, want %v", got, want)
	}
}

func TestM2LinesOnlyOccurringExtensions(t *testing.T) {
	root := writeFixture(t, map[string]string{"a.md": "a\nb\n"})
	got, err := Lines(root)
	if err != nil {
		t.Fatal(err)
	}
	if !sameCounts(got, map[string]int{".md": 2}) {
		t.Fatalf("Lines(md only) = %v, want only .md: 2", got)
	}
}
