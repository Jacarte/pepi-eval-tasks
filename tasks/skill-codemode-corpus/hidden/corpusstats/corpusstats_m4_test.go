package corpusstats

import (
	"testing"
)

func checkTop(t *testing.T, what string, got, want []FileTokens) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %d entries %v, want %d entries %v", what, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: entry %d = %+v, want %+v", what, i, got[i], want[i])
		}
	}
}

func TestM4TopMatchesReferenceOnCorpus(t *testing.T) {
	all := refTop(t, m1Corpus)
	want := all[:10]
	got, err := Top(m1Corpus, 10)
	if err != nil {
		t.Fatal(err)
	}
	checkTop(t, "Top(corpus, 10)", got, want)
}

func TestM4TopCountsAndAllFiles(t *testing.T) {
	all := refTop(t, m1Corpus)
	if got, err := Top(m1Corpus, 0); err != nil || len(got) != 0 {
		t.Fatalf("Top(corpus, 0) = %v, %v; want no entries", got, err)
	}
	got, err := Top(m1Corpus, len(all)+5)
	if err != nil {
		t.Fatal(err)
	}
	checkTop(t, "Top(corpus, all)", got, all)
}

func TestM4TopFixtureTiesAndZero(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"b.txt":   "two words\n",               // 2
		"a/z.md":  "two words\n",               // 2, path a/z.md sorts after a.go
		"a.go":    "// no\nx y\n",              // 2 (the comment line is skipped)
		"c.txt":   "one two three four five\n", // 5
		"zero.md": "# only a comment\n",        // 0, still listed
		"n.json":  "many many many\n",          // not counted
	})
	got, err := Top(root, 10)
	if err != nil {
		t.Fatal(err)
	}
	checkTop(t, "Top(fixture, 10)", got, []FileTokens{
		{Path: "c.txt", Tokens: 5},
		{Path: "a.go", Tokens: 2},
		{Path: "a/z.md", Tokens: 2},
		{Path: "b.txt", Tokens: 2},
		{Path: "zero.md", Tokens: 0},
	})
	got, err = Top(root, 2)
	if err != nil {
		t.Fatal(err)
	}
	checkTop(t, "Top(fixture, 2)", got, []FileTokens{
		{Path: "c.txt", Tokens: 5},
		{Path: "a.go", Tokens: 2},
	})
}
