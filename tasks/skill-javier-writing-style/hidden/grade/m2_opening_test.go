package grade

import (
	"regexp"
	"strings"
	"testing"
)

// The opening paragraph names a pain point and its impact, and does not announce the solution.
var (
	openingPain     = regexp.MustCompile(`(?i)slow|lost|losing|\bwait|ephemeral|from scratch|again|bottleneck|pain|throw|discard|empty|miss`)
	openingImpact   = regexp.MustCompile(`\d|(?i)cost|\$|usd|developer|engineer|feedback|throughput|per (week|month|day)|minutes?|seconds?|hours?`)
	openingAnnounce = regexp.MustCompile(`(?i)in this (post|article)|this (post|article) (presents|describes|shows|explains)|\b(we|i) (built|introduce|introduced|developed|created)\b`)
)

func TestM2Opening(t *testing.T) {
	para := strings.Join(firstParagraph(t, readPost(t)), " ")
	if m := openingAnnounce.FindString(para); m != "" {
		t.Fatalf("the opening paragraph announces the solution (%q): %s", m, para)
	}
	if !openingPain.MatchString(para) {
		t.Fatalf("the opening paragraph names no pain point: %s", para)
	}
	if !openingImpact.MatchString(para) {
		t.Fatalf("the opening paragraph names no impact (cost, time or people affected): %s", para)
	}
}
