package grade

import (
	"regexp"
	"testing"
)

// hedges are pragmatic phrases the author uses to admit limits. The post must use at least
// minHedges distinct ones.
var hedges = []struct {
	name string
	re   *regexp.Regexp
}{
	{"to the best of my knowledge", regexp.MustCompile(`(?i)to the best of (my|our) knowledge`)},
	{"this is not foolproof", regexp.MustCompile(`(?i)not foolproof`)},
	{"there exists a trade-off", regexp.MustCompile(`(?i)there exists a trade-?off`)},
	{"unfortunately", regexp.MustCompile(`(?i)\bunfortunately\b`)},
	{"speaking anecdotally", regexp.MustCompile(`(?i)speaking anecdotally`)},
	{"we chose to", regexp.MustCompile(`(?i)\b(we|i) chose to\b`)},
}

const minHedges = 3

func TestM3Voice(t *testing.T) {
	post := readPost(t)
	var found []string
	for _, h := range hedges {
		if h.re.MatchString(post) {
			found = append(found, h.name)
		}
	}
	if len(found) < minHedges {
		t.Fatalf("the post uses %d pragmatic phrases, want at least %d; found %q", len(found), minHedges, found)
	}
}
