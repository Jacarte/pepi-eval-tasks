package grade

import (
	"regexp"
	"testing"
)

// sectionKinds are matched against heading text. Each kind's first heading must come after the
// previous kind's, so the post reads problem, solution, results, discussion.
var sectionKinds = []struct {
	name string
	re   *regexp.Regexp
}{
	{"problem or why existing fails", regexp.MustCompile(`(?i)problem|why|motivation|background|context|challenge|pain`)},
	{"solution", regexp.MustCompile(`(?i)solution|approach|implementation|setup|design|architecture|the fix|what we built`)},
	{"results", regexp.MustCompile(`(?i)result|benchmark|measure|evaluation|outcome|before and after|numbers`)},
	{"discussion or limitations", regexp.MustCompile(`(?i)discussion|limitation|trade-?off|caveat|drawback|alternative`)},
}

func TestM1Structure(t *testing.T) {
	heads := headings(readPost(t))
	last := -1
	for _, kind := range sectionKinds {
		idx := -1
		for i, h := range heads {
			if kind.re.MatchString(h) {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatalf("no %s heading; headings are %q", kind.name, heads)
		}
		if idx <= last {
			t.Fatalf("the %s heading %q comes before the section before it; headings are %q", kind.name, heads[idx], heads)
		}
		last = idx
	}
}
