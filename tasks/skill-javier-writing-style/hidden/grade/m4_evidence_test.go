package grade

import (
	"regexp"
	"testing"
)

// The before and after numbers from notes.md, and a `>` aside or caveat.
var evidenceNumbers = []struct {
	name string
	re   *regexp.Regexp
}{
	{"median before 412 s", regexp.MustCompile(`\b412\b`)},
	{"median after 118 s", regexp.MustCompile(`\b118\b`)},
	{"cost before 78.72", regexp.MustCompile(`\b78\.72\b`)},
	{"cost after 24.96", regexp.MustCompile(`\b24\.96\b`)},
}

var blockquoteLine = regexp.MustCompile(`(?m)^>`)

func TestM4Evidence(t *testing.T) {
	post := readPost(t)
	for _, n := range evidenceNumbers {
		if !n.re.MatchString(post) {
			t.Errorf("the post does not quote the %s", n.name)
		}
	}
	if !blockquoteLine.MatchString(post) {
		t.Errorf("the post has no > blockquote aside or caveat")
	}
}
