package corpusstats

import (
	"testing"
)

func TestM3TokensMatchesReferenceOnCorpus(t *testing.T) {
	want := 0
	for _, f := range refWalk(t, m1Corpus) {
		want += refTokens(f.ext, f.data)
	}
	got, err := Tokens(m1Corpus)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Tokens(corpus) = %d, want %d", got, want)
	}
}

func TestM3TokensFixture(t *testing.T) {
	root := writeFixture(t, map[string]string{
		// "Don't stop—it's 2 o'clock!" -> don't, stop, it's, o'clock (the em dash and the digit are
		// not token characters); the two "#" lines are comments; "Café naïve" -> caf, na, ve.
		"a.txt": "Don't stop\u2014it's 2 o'clock!\n# comment words here\n   # indented comment\nCaf\u00e9 na\u00efve\n",
		// "// hidden words" is a comment; "/* block words */" is not (block, words); the code line
		// has x, foo, bar, baz.
		"b.go": "// hidden words\n/* block words */\n\tx := foo(\"bar\") + baz\n",
		// Markdown headings start with '#', so both lines are comments; "plain text" counts.
		"c.md": "# Heading words\n## Sub\nplain text\n",
		// Apostrophe-only runs are tokens: "''" and "''".
		"d.txt": "'' ''\n",
		// CRLF endings: alpha, beta, delta; "# gamma" is a comment.
		"e.txt": "alpha beta\r\n# gamma\r\nDelta",
		// Not counted at all.
		"NOTES.TXT": "ignored words here\n",
	})
	got, err := Tokens(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != 20 {
		t.Fatalf("Tokens(fixture) = %d, want 20", got)
	}
}

func TestM3TokensEmptyTree(t *testing.T) {
	got, err := Tokens(writeFixture(t, map[string]string{"x.json": "many words here\n"}))
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Fatalf("Tokens(no counted files) = %d, want 0", got)
	}
}
