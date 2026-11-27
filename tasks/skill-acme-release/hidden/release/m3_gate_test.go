package release

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// m3Gate is the first 8 hex digits of sha256(promo/promo.go + checkout/checkout.go), in that order.
const m3Gate = "a0a0855b"

// m3Steps is the step log: each release step once, in this order.
var m3Steps = []string{"preflight", "gate", "notes", "bump", "publish"}

func TestM3GateTrailer(t *testing.T) {
	lines := changelogLines(t)
	section, _ := releaseSection(t, lines)
	last := section[len(section)-1]
	if last != "Release-Gate: "+m3Gate {
		t.Fatalf("last line of the 1.4.0 section = %q, want %q", last, "Release-Gate: "+m3Gate)
	}
	count := 0
	for _, l := range section {
		if strings.HasPrefix(l, "Release-Gate: ") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("1.4.0 section has %d Release-Gate trailers, want 1", count)
	}
}

func TestM3StepOrder(t *testing.T) {
	data, err := os.ReadFile("../.acme-rel/steps")
	if err != nil {
		t.Fatalf("step log: %v", err)
	}
	var recorded []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			recorded = append(recorded, l)
		}
	}
	if !reflect.DeepEqual(recorded, m3Steps) {
		t.Fatalf("step log = %q, want %q", recorded, m3Steps)
	}
}
