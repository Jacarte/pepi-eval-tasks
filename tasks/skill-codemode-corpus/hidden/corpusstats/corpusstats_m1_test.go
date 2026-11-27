package corpusstats

import (
	"testing"
)

const m1Corpus = "../corpus"

func TestM1FilesMatchesReferenceOnCorpus(t *testing.T) {
	want := map[string]int{}
	for _, f := range refWalk(t, m1Corpus) {
		want[f.ext]++
	}
	got, err := Files(m1Corpus)
	if err != nil {
		t.Fatal(err)
	}
	if !sameCounts(got, want) {
		t.Fatalf("Files(corpus) = %v, want %v", got, want)
	}
}

func TestM1FilesFixture(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"a.txt":             "x\n",
		"sub/b.md":          "y\n",
		"sub/deep/c.go":     "package p\n",
		"sub/deep/d.go":     "",
		"NOTES.TXT":         "not counted: uppercase extension\n",
		"x.txt.bak":         "not counted: suffix\n",
		"y.json":            "{}\n",
		"sub/deep/z.md.png": "not counted\n",
	})
	got, err := Files(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{".txt": 1, ".md": 1, ".go": 2}
	if !sameCounts(got, want) {
		t.Fatalf("Files(fixture) = %v, want %v", got, want)
	}
}

func TestM1FilesNoCountedFiles(t *testing.T) {
	root := writeFixture(t, map[string]string{"only.json": "{}\n", "a/b.TXT": "x\n"})
	got, err := Files(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("Files(no counted files) = %v, want no entries", got)
	}
}

func TestM1FilesMissingRootIsError(t *testing.T) {
	if _, err := Files(t.TempDir() + "/does-not-exist"); err == nil {
		t.Fatal("Files on a missing root returned no error")
	}
}
