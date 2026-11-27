package release

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// m1Section is the 1.4.0 section without its gate trailer (checked by TestM3).
var m1Section = []string{
	"## acme 1.4.0",
	"- [ACME-31] feat(checkout): Show applied promo codes in quote",
	"- [ACME-12] fix(promo): Keep entered order when codes combine",
}

var m1Legacy = []string{
	"## 1.3.0 - 2026-10-02",
	"- fix auth token refresh (ACME-7)",
	"- feat quote cache for carts (ACME-9)",
}

// changelogLines returns CHANGELOG.md, one entry per line, trailing blank lines dropped.
func changelogLines(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile("../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(data), "\n"), "\n")
}

// releaseSection returns the lines of the 1.4.0 section (up to the next "## " heading, blank lines
// dropped) and the lines from the next "## " heading on.
func releaseSection(t *testing.T, lines []string) (section, rest []string) {
	t.Helper()
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "## ") {
			if start == -1 {
				start = i
				continue
			}
			return nonBlank(lines[start:i]), lines[i:]
		}
	}
	if start == -1 {
		t.Fatal("CHANGELOG.md has no release section")
	}
	return nonBlank(lines[start:]), nil
}

func nonBlank(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func TestM1ChangelogSection(t *testing.T) {
	lines := changelogLines(t)
	if lines[0] != "# Changelog" || lines[1] != "" || !strings.HasPrefix(lines[2], "## acme 1.4.0") {
		t.Fatalf("CHANGELOG.md must open with the title, a blank line, then the 1.4.0 section; got %q %q %q", lines[0], lines[1], lines[2])
	}
	section, rest := releaseSection(t, lines)
	var got []string
	for _, l := range section {
		if !strings.HasPrefix(l, "Release-Gate: ") {
			got = append(got, l)
		}
	}
	if !reflect.DeepEqual(got, m1Section) {
		t.Fatalf("1.4.0 section = %q, want %q", got, m1Section)
	}
	if !reflect.DeepEqual(rest, m1Legacy) {
		t.Fatalf("older sections changed: %q, want %q", rest, m1Legacy)
	}
}
